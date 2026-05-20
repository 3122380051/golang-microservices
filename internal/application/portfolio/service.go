package portfolio

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
	"github.com/3122380051/golang-microservices/internal/infrastructure/broker"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Service orchestrates portfolio operations and event handling
type Service struct {
	logger                *slog.Logger
	portfolioRepository   domain.PortfolioRepository
	tradeRepository       domain.TradeResultRepository
	producer              *broker.KafkaProducer
	consumer              *broker.KafkaConsumer
	calculator            *PnLCalculator
	priceFetcher          PriceFetcher
	processedExecutionsMu sync.RWMutex
	processedExecutions   map[string]bool // executionID -> processed
	portfoliosCacheMu     sync.RWMutex
	portfoliosCache       map[string]*domain.Portfolio // userID -> Portfolio
}

// ExecutionFilledEvent represents an execution.filled event from Kafka
type ExecutionFilledEvent struct {
	EventID          string
	ExecutionID      string
	OrderID          string
	ClientOrderID    string
	CorrelationID    string
	UserID           string
	Symbol           string
	Side             string
	OriginalQty      float64
	ExecutedQty      float64
	AverageFillPrice float64
	Fees             float64
	Status           string
	EventType        string
	Timestamp        time.Time
	TraceID          string
}

// PriceFetcher interface for getting current prices
type PriceFetcher interface {
	GetPrice(ctx context.Context, symbol string) (float64, error)
	GetPrices(ctx context.Context, symbols []string) (map[string]float64, error)
}

// NewService creates a new portfolio service
func NewService(
	logger *slog.Logger,
	portfolioRepo domain.PortfolioRepository,
	tradeRepo domain.TradeResultRepository,
	producer *broker.KafkaProducer,
	consumer *broker.KafkaConsumer,
	calculator *PnLCalculator,
	priceFetcher PriceFetcher,
) *Service {
	return &Service{
		logger:              logger,
		portfolioRepository: portfolioRepo,
		tradeRepository:     tradeRepo,
		producer:            producer,
		consumer:            consumer,
		calculator:          calculator,
		priceFetcher:        priceFetcher,
		processedExecutions: make(map[string]bool),
		portfoliosCache:     make(map[string]*domain.Portfolio),
	}
}

// ConsumeExecutionFilled listens for execution.filled events from Kafka
func (s *Service) ConsumeExecutionFilled(ctx context.Context) {
	if s.consumer == nil {
		s.logger.Warn("kafka consumer not initialized, skipping execution consumption")
		return
	}

	s.logger.Info("starting execution.filled event consumer")

	handler := func(ctx context.Context, msg kafka.Message) error {
		return s.handleExecutionFilled(ctx, msg)
	}

	if err := s.consumer.Consume(ctx, handler); err != nil {
		s.logger.Error("consumer error", "error", err)
	}
}

func (s *Service) handleExecutionFilled(ctx context.Context, msg kafka.Message) error {
	var event ExecutionFilledEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal execution event: %w", err)
	}

	// Idempotency check
	s.processedExecutionsMu.RLock()
	processed := s.processedExecutions[event.ExecutionID]
	s.processedExecutionsMu.RUnlock()

	if processed {
		s.logger.Debug("execution already processed", "execution_id", event.ExecutionID)
		return nil
	}

	// Mark as processed
	s.processedExecutionsMu.Lock()
	s.processedExecutions[event.ExecutionID] = true
	s.processedExecutionsMu.Unlock()

	// Get or create portfolio
	portfolio, err := s.getOrCreatePortfolio(ctx, event.UserID)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	// Update portfolio with filled execution
	if err := s.updatePortfolioWithExecution(ctx, portfolio, &event); err != nil {
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	// Persist updated portfolio
	if err := s.portfolioRepository.Update(ctx, portfolio); err != nil {
		return fmt.Errorf("failed to persist portfolio: %w", err)
	}

	// Update cache
	s.portfoliosCacheMu.Lock()
	s.portfoliosCache[portfolio.UserID] = portfolio
	s.portfoliosCacheMu.Unlock()

	// Publish portfolio updated event if PnL realized
	if event.ExecutedQty > 0 {
		s.publishPortfolioEvent(portfolio, &event)
	}

	return nil
}

func (s *Service) getOrCreatePortfolio(ctx context.Context, userID string) (*domain.Portfolio, error) {
	// Check cache first
	s.portfoliosCacheMu.RLock()
	cached, exists := s.portfoliosCache[userID]
	s.portfoliosCacheMu.RUnlock()

	if exists && cached != nil {
		return cached, nil
	}

	// Try to fetch from repository
	portfolio, err := s.portfolioRepository.GetByUserID(ctx, userID)
	if err == nil && portfolio != nil {
		s.portfoliosCacheMu.Lock()
		s.portfoliosCache[userID] = portfolio
		s.portfoliosCacheMu.Unlock()
		return portfolio, nil
	}

	// Create new portfolio with default balance
	portfolio = domain.NewPortfolio(userID, 10000.0) // Default 10k balance
	if err := s.portfolioRepository.Create(ctx, portfolio); err != nil {
		return nil, fmt.Errorf("failed to create portfolio: %w", err)
	}

	s.portfoliosCacheMu.Lock()
	s.portfoliosCache[userID] = portfolio
	s.portfoliosCacheMu.Unlock()

	return portfolio, nil
}

func (s *Service) updatePortfolioWithExecution(ctx context.Context, portfolio *domain.Portfolio, event *ExecutionFilledEvent) error {
	if event.ExecutedQty == 0 {
		return nil
	}

	// Convert side string to OrderSide
	var side domain.OrderSide
	switch event.Side {
	case "BUY":
		side = domain.OrderSideBuy
	case "SELL":
		side = domain.OrderSideSell
	default:
		return fmt.Errorf("unknown order side: %s", event.Side)
	}

	// Check if position already exists
	existing, hasPosition := portfolio.Positions[event.Symbol]

	if !hasPosition {
		// Create new position
		position := domain.NewPosition(event.Symbol, side, event.ExecutedQty, event.AverageFillPrice)
		return portfolio.AddPosition(position)
	}

	// Update existing position
	if existing.Side == side {
		// Adding to position (pyramid or average down)
		totalQty := existing.Quantity + event.ExecutedQty
		totalCost := (existing.AverageEntryPrice * existing.Quantity) + (event.AverageFillPrice * event.ExecutedQty)
		existing.AverageEntryPrice = totalCost / totalQty
		existing.Quantity = totalQty
		existing.UpdatePrice(event.AverageFillPrice)
	} else {
		// Opposite side = reducing or closing position
		newQty := existing.Quantity - event.ExecutedQty

		if newQty > 0 {
			// Partial close
			existing.Quantity = newQty
			existing.UpdatePrice(event.AverageFillPrice)
		} else if newQty == 0 {
			// Position fully closed - realize PnL
			pnl := s.calculator.CalculateClosedPnL(existing, event.AverageFillPrice)
			if err := portfolio.RealizePnL(pnl, event.Fees); err != nil {
				return err
			}

			// Save trade result
			tradeResult := &domain.TradeResult{
				ID:          uuid.New().String(),
				ExecutionID: event.ExecutionID,
				Symbol:      event.Symbol,
				Side:        existing.Side,
				EntryPrice:  existing.AverageEntryPrice,
				ExitPrice:   event.AverageFillPrice,
				Quantity:    existing.Quantity,
				RealizedPnL: pnl,
				Fees:        event.Fees,
				NetPnL:      pnl - event.Fees,
				EntryTime:   time.Now().Add(-time.Hour), // TODO: get from order
				ExitTime:    time.Now(),
				IsWin:       (pnl - event.Fees) > 0,
			}
			if err := s.tradeRepository.Create(ctx, tradeResult); err != nil {
				s.logger.Error("failed to save trade result", "error", err)
			}

			// Remove closed position
			return portfolio.RemovePosition(event.Symbol)
		} else {
			// Flip side = existing position closes + new position in opposite direction
			closedPnL := s.calculator.CalculateClosedPnL(existing, event.AverageFillPrice)
			if err := portfolio.RealizePnL(closedPnL, 0); err != nil {
				return err
			}

			// Create new position on opposite side
			newPosition := domain.NewPosition(event.Symbol, side, -newQty, event.AverageFillPrice)
			if err := portfolio.RemovePosition(event.Symbol); err != nil {
				return err
			}
			return portfolio.AddPosition(newPosition)
		}
	}

	return nil
}

func (s *Service) publishPortfolioEvent(portfolio *domain.Portfolio, execEvent *ExecutionFilledEvent) {
	if s.producer == nil {
		return
	}

	event := &domain.PnLEvent{
		EventID:           uuid.New().String(),
		UserID:            portfolio.UserID,
		ExecutionID:       execEvent.ExecutionID,
		CorrelationID:     execEvent.CorrelationID,
		Symbol:            execEvent.Symbol,
		Side:              domain.OrderSideBuy, // TODO: parse from execEvent.Side
		Quantity:          execEvent.ExecutedQty,
		EntryPrice:        0, // TODO: get from position history
		ExitPrice:         execEvent.AverageFillPrice,
		RealizedPnL:       portfolio.RealizedPnL,
		Fees:              execEvent.Fees,
		NetPnL:            portfolio.RealizedPnL - execEvent.Fees,
		TotalPortfolioPnL: portfolio.TotalPnL,
		Timestamp:         time.Now(),
		EventType:         "portfolio.pnl_updated",
		TraceID:           execEvent.TraceID,
	}

	if err := s.producer.PublishJSON(context.Background(), "portfolio.updated", execEvent.ExecutionID, event); err != nil {
		s.logger.Error("failed to publish portfolio event", "error", err)
	}
}

// Query methods

// GetPortfolio retrieves a portfolio by user ID
func (s *Service) GetPortfolio(ctx context.Context, userID string) (*domain.Portfolio, error) {
	return s.portfolioRepository.GetByUserID(ctx, userID)
}

// ListPortfolios lists all portfolios
func (s *Service) ListPortfolios(ctx context.Context) ([]domain.Portfolio, error) {
	return s.portfolioRepository.ListAll(ctx)
}

// GetTradeHistory retrieves trade results for a user
func (s *Service) GetTradeHistory(ctx context.Context, userID string) ([]domain.TradeResult, error) {
	return s.tradeRepository.ListByUser(ctx, userID)
}

// UpdatePortfolioPrices updates all positions with current prices (for mark-to-market)
func (s *Service) UpdatePortfolioPrices(ctx context.Context, userID string, prices map[string]float64) error {
	portfolio, err := s.GetPortfolio(ctx, userID)
	if err != nil {
		return err
	}

	for symbol, price := range prices {
		if position, exists := portfolio.Positions[symbol]; exists {
			position.UpdatePrice(price)
		}
	}

	portfolio.UpdatedAt = time.Now()
	portfolio.RecalculateTotals()

	return s.portfolioRepository.Update(ctx, portfolio)
}
