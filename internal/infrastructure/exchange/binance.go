package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/3122380051/golang-microservices/internal/domain"
)

const defaultBinanceBaseURL = "https://api.binance.com"

// BinanceClient implements Adapter via Binance public REST APIs.
type BinanceClient struct {
	baseURL string
	http    *http.Client
}

func NewBinanceClient() *BinanceClient {
	return NewBinanceClientWithBaseURL(defaultBinanceBaseURL, 5*time.Second)
}

func NewBinanceClientWithBaseURL(baseURL string, timeout time.Duration) *BinanceClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBinanceBaseURL
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &BinanceClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

type binanceBookTicker struct {
	Symbol string `json:"symbol"`
	Bid    string `json:"bidPrice"`
	Ask    string `json:"askPrice"`
}

func (b *BinanceClient) GetTicker(ctx context.Context, symbol string) (domain.MarketPrice, error) {
	u := b.baseURL + "/api/v3/ticker/bookTicker?symbol=" + url.QueryEscape(strings.ToUpper(strings.TrimSpace(symbol)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return domain.MarketPrice{}, err
	}

	resp, err := b.http.Do(req)
	if err != nil {
		return domain.MarketPrice{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return domain.MarketPrice{}, fmt.Errorf("binance ticker status: %d", resp.StatusCode)
	}

	var raw binanceBookTicker
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return domain.MarketPrice{}, err
	}

	bid, err := strconv.ParseFloat(raw.Bid, 64)
	if err != nil {
		return domain.MarketPrice{}, err
	}
	ask, err := strconv.ParseFloat(raw.Ask, 64)
	if err != nil {
		return domain.MarketPrice{}, err
	}

	return domain.MarketPrice{
		Symbol:   raw.Symbol,
		Exchange: "binance",
		Price:    (bid + ask) / 2,
		Bid:      bid,
		Ask:      ask,
		Ts:       time.Now().UTC(),
	}, nil
}

func (b *BinanceClient) GetCandles(ctx context.Context, symbol, interval string, limit int) ([]domain.Candle, error) {
	if limit <= 0 {
		limit = 100
	}
	u := fmt.Sprintf("%s/api/v3/klines?symbol=%s&interval=%s&limit=%d",
		b.baseURL,
		url.QueryEscape(strings.ToUpper(strings.TrimSpace(symbol))),
		url.QueryEscape(interval),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("binance candles status: %d", resp.StatusCode)
	}

	var rows [][]any
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}

	candles := make([]domain.Candle, 0, len(rows))
	for _, row := range rows {
		if len(row) < 7 {
			continue
		}

		openTs, _ := toInt64(row[0])
		open, _ := toFloat64(row[1])
		high, _ := toFloat64(row[2])
		low, _ := toFloat64(row[3])
		closeV, _ := toFloat64(row[4])
		volume, _ := toFloat64(row[5])
		closeTs, _ := toInt64(row[6])

		candles = append(candles, domain.Candle{
			Symbol:    strings.ToUpper(strings.TrimSpace(symbol)),
			Exchange:  "binance",
			Interval:  interval,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closeV,
			Volume:    volume,
			OpenTime:  time.UnixMilli(openTs).UTC(),
			CloseTime: time.UnixMilli(closeTs).UTC(),
		})
	}

	return candles, nil
}

type binanceDepth struct {
	Bids [][]string `json:"bids"`
	Asks [][]string `json:"asks"`
}

func (b *BinanceClient) GetOrderBook(ctx context.Context, symbol string, limit int) (domain.OrderBook, error) {
	if limit <= 0 {
		limit = 20
	}
	u := fmt.Sprintf("%s/api/v3/depth?symbol=%s&limit=%d",
		b.baseURL,
		url.QueryEscape(strings.ToUpper(strings.TrimSpace(symbol))),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return domain.OrderBook{}, err
	}

	resp, err := b.http.Do(req)
	if err != nil {
		return domain.OrderBook{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return domain.OrderBook{}, fmt.Errorf("binance order book status: %d", resp.StatusCode)
	}

	var raw binanceDepth
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return domain.OrderBook{}, err
	}

	ob := domain.OrderBook{
		Symbol:   strings.ToUpper(strings.TrimSpace(symbol)),
		Exchange: "binance",
		Bids:     toLevels(raw.Bids),
		Asks:     toLevels(raw.Asks),
		Ts:       time.Now().UTC(),
	}
	return ob, nil
}

func toLevels(input [][]string) []domain.OrderLevel {
	out := make([]domain.OrderLevel, 0, len(input))
	for _, item := range input {
		if len(item) < 2 {
			continue
		}
		p, errP := strconv.ParseFloat(item[0], 64)
		q, errQ := strconv.ParseFloat(item[1], 64)
		if errP != nil || errQ != nil {
			continue
		}
		out = append(out, domain.OrderLevel{Price: p, Qty: q})
	}
	return out
}

func toFloat64(v any) (float64, bool) {
	s, ok := v.(string)
	if ok {
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	}
	f, ok := v.(float64)
	return f, ok
}

func toInt64(v any) (int64, bool) {
	switch vv := v.(type) {
	case float64:
		return int64(vv), true
	case int64:
		return vv, true
	case json.Number:
		i, err := vv.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}
