package risk

import (
	"fmt"
	"log/slog"

	"github.com/3122380051/golang-microservices/internal/domain"
)

// Evaluator performs risk assessment and decision making
type Evaluator struct {
	logger *slog.Logger
}

// NewEvaluator creates a new risk evaluator
func NewEvaluator(logger *slog.Logger) *Evaluator {
	return &Evaluator{
		logger: logger,
	}
}

// EvaluateSignal evaluates a strategy signal against risk policies
// Returns a RiskDecision with approval status and detailed checks
func (e *Evaluator) EvaluateSignal(
	policy *domain.RiskPolicy,
	portfolio *domain.PortfolioSnapshot,
	signal *domain.Signal,
	estimatedPrice float64,
) *domain.RiskDecision {

	decision := &domain.RiskDecision{
		SignalID:       signal.ID,
		UserID:         signal.UserID,
		StrategyID:     signal.StrategyID,
		Symbol:         signal.Symbol,
		Side:           signal.Side,
		Quantity:       signal.Quantity,
		EstimatedPrice: estimatedPrice,
		IsApproved:     true,
		Checks: domain.ChecksDetail{
			PositionSizeCheck: domain.CheckResult{Passed: true},
			LeverageCheck:     domain.CheckResult{Passed: true},
			MarginCheck:       domain.CheckResult{Passed: true},
			DailyLossCheck:    domain.CheckResult{Passed: true},
			ExposureCheck:     domain.CheckResult{Passed: true},
		},
	}

	// Check 1: Position Size
	if !e.checkPositionSize(policy, signal, estimatedPrice, decision) {
		decision.IsApproved = false
		decision.RejectionReason = decision.Checks.PositionSizeCheck.Reason
		return decision
	}

	// Check 2: Leverage
	if !e.checkLeverage(policy, portfolio, signal, estimatedPrice, decision) {
		decision.IsApproved = false
		decision.RejectionReason = decision.Checks.LeverageCheck.Reason
		return decision
	}

	// Check 3: Margin Ratio
	if !e.checkMarginRatio(policy, portfolio, decision) {
		decision.IsApproved = false
		decision.RejectionReason = decision.Checks.MarginCheck.Reason
		return decision
	}

	// Check 4: Daily Loss
	if !e.checkDailyLoss(policy, portfolio, decision) {
		decision.IsApproved = false
		decision.RejectionReason = decision.Checks.DailyLossCheck.Reason
		return decision
	}

	// Check 5: Total Exposure
	if !e.checkExposure(policy, portfolio, signal, estimatedPrice, decision) {
		decision.IsApproved = false
		decision.RejectionReason = decision.Checks.ExposureCheck.Reason
		return decision
	}

	return decision
}

// checkPositionSize validates order quantity doesn't exceed max_position_size
func (e *Evaluator) checkPositionSize(
	policy *domain.RiskPolicy,
	signal *domain.Signal,
	price float64,
	decision *domain.RiskDecision,
) bool {
	orderValue := signal.Quantity * price
	check := &decision.Checks.PositionSizeCheck

	if orderValue > policy.MaxPositionSize {
		check.Passed = false
		check.Reason = fmt.Sprintf(
			"position size %.4f %s = $%.2f exceeds max $%.2f",
			signal.Quantity, signal.Symbol, orderValue, policy.MaxPositionSize,
		)
		check.Value = fmt.Sprintf("$%.2f / $%.2f", orderValue, policy.MaxPositionSize)
		return false
	}

	check.Passed = true
	check.Reason = fmt.Sprintf("position size OK: $%.2f <= $%.2f", orderValue, policy.MaxPositionSize)
	check.Value = fmt.Sprintf("$%.2f", orderValue)
	return true
}

// checkLeverage validates leverage doesn't exceed max_leverage
func (e *Evaluator) checkLeverage(
	policy *domain.RiskPolicy,
	portfolio *domain.PortfolioSnapshot,
	signal *domain.Signal,
	price float64,
	decision *domain.RiskDecision,
) bool {
	orderValue := signal.Quantity * price
	check := &decision.Checks.LeverageCheck

	// Calculate projected leverage after order
	// leverage = (current_exposure + new_order_value) / account_equity
	accountEquity := portfolio.TotalBalance
	newExposure := portfolio.TotalExposure + orderValue
	projectedLeverage := newExposure / accountEquity

	if projectedLeverage > policy.MaxLeverage {
		check.Passed = false
		check.Reason = fmt.Sprintf(
			"projected leverage %.2fx exceeds max %.2fx (exposure: $%.2f / equity: $%.2f)",
			projectedLeverage, policy.MaxLeverage, newExposure, accountEquity,
		)
		check.Value = fmt.Sprintf("%.2fx / %.2fx", projectedLeverage, policy.MaxLeverage)
		return false
	}

	check.Passed = true
	check.Reason = fmt.Sprintf("leverage OK: %.2fx <= %.2fx", projectedLeverage, policy.MaxLeverage)
	check.Value = fmt.Sprintf("%.2fx", projectedLeverage)
	return true
}

// checkMarginRatio validates available margin percentage
func (e *Evaluator) checkMarginRatio(
	policy *domain.RiskPolicy,
	portfolio *domain.PortfolioSnapshot,
	decision *domain.RiskDecision,
) bool {
	check := &decision.Checks.MarginCheck

	if portfolio.TotalBalance == 0 {
		check.Passed = false
		check.Reason = "insufficient balance for margin check"
		return false
	}

	marginRatio := portfolio.AvailableMargin / portfolio.TotalBalance
	if marginRatio < policy.MinMarginRatio {
		check.Passed = false
		check.Reason = fmt.Sprintf(
			"margin ratio %.1f%% below minimum %.1f%% (available: $%.2f / total: $%.2f)",
			marginRatio*100, policy.MinMarginRatio*100, portfolio.AvailableMargin, portfolio.TotalBalance,
		)
		check.Value = fmt.Sprintf("%.1f%% / %.1f%%", marginRatio*100, policy.MinMarginRatio*100)
		return false
	}

	check.Passed = true
	check.Reason = fmt.Sprintf("margin ratio OK: %.1f%% >= %.1f%%", marginRatio*100, policy.MinMarginRatio*100)
	check.Value = fmt.Sprintf("%.1f%%", marginRatio*100)
	return true
}

// checkDailyLoss validates daily loss doesn't exceed limit
func (e *Evaluator) checkDailyLoss(
	policy *domain.RiskPolicy,
	portfolio *domain.PortfolioSnapshot,
	decision *domain.RiskDecision,
) bool {
	check := &decision.Checks.DailyLossCheck

	if portfolio.RealizedPnL < policy.MaxDailyLoss {
		check.Passed = false
		check.Reason = fmt.Sprintf(
			"daily loss $%.2f exceeds limit $%.2f (stop-out triggered)",
			portfolio.RealizedPnL, policy.MaxDailyLoss,
		)
		check.Value = fmt.Sprintf("$%.2f / $%.2f", portfolio.RealizedPnL, policy.MaxDailyLoss)
		return false
	}

	check.Passed = true
	check.Reason = fmt.Sprintf("daily loss OK: $%.2f >= $%.2f", portfolio.RealizedPnL, policy.MaxDailyLoss)
	check.Value = fmt.Sprintf("$%.2f", portfolio.RealizedPnL)
	return true
}

// checkExposure validates total exposure doesn't exceed limit
func (e *Evaluator) checkExposure(
	policy *domain.RiskPolicy,
	portfolio *domain.PortfolioSnapshot,
	signal *domain.Signal,
	price float64,
	decision *domain.RiskDecision,
) bool {
	orderValue := signal.Quantity * price
	check := &decision.Checks.ExposureCheck

	newTotalExposure := portfolio.TotalExposure + orderValue
	if newTotalExposure > policy.MaxExposure {
		check.Passed = false
		check.Reason = fmt.Sprintf(
			"total exposure $%.2f exceeds limit $%.2f",
			newTotalExposure, policy.MaxExposure,
		)
		check.Value = fmt.Sprintf("$%.2f / $%.2f", newTotalExposure, policy.MaxExposure)
		return false
	}

	check.Passed = true
	check.Reason = fmt.Sprintf("exposure OK: $%.2f <= $%.2f", newTotalExposure, policy.MaxExposure)
	check.Value = fmt.Sprintf("$%.2f", newTotalExposure)
	return true
}
