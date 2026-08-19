package spot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/k4k3ru-hub/mexc/go/rest"
	"github.com/k4k3ru-hub/mexc/go/rest/transport"
)

const DefaultBaseURL = "https://api.mexc.com"

var spotSymbolPattern = regexp.MustCompile(`^[A-Z0-9]+$`)

// Decimal preserves a JSON string or number lexeme without float conversion.
type Decimal string

// UnmarshalJSON decodes a decimal string or number.
//
// Version:
//   - 2026-08-19: Added.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("failed to decode decimal: destination=null")
	}
	if string(data) == "null" {
		*d = ""
		return nil
	}
	var s string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("failed to decode decimal: %w", err)
		}
	} else {
		s = string(data)
	}
	*d = Decimal(s)
	return nil
}

// Client provides composed Spot V3 public market-data operations.
type Client struct{ executor transport.Executor }

// NewClient creates a Spot REST client without making a network request.
//
// Version:
//   - 2026-08-19: Added.
func NewClient(option *rest.ClientOption) (*Client, error) {
	executor, err := rest.NewClient(option, DefaultBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create spot rest client: %w", err)
	}
	return NewClientWithExecutor(executor)
}

// NewClientWithExecutor creates a Spot REST client with an injected executor.
//
// Version:
//   - 2026-08-19: Added.
func NewClientWithExecutor(executor transport.Executor) (*Client, error) {
	if executor == nil {
		return nil, fmt.Errorf("failed to create spot rest client: executor=null")
	}
	return &Client{executor: executor}, nil
}

type ExchangeInfoParams struct {
	Symbol        string
	Symbols       []string
	Status        string
	TradeSideType string
}
type RateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int64  `json:"intervalNum"`
	Limit         int64  `json:"limit"`
}
type ExchangeFilter map[string]json.RawMessage
type SymbolFilter map[string]json.RawMessage
type SymbolInfo struct {
	Symbol                     string         `json:"symbol"`
	Status                     string         `json:"status"`
	BaseAsset                  string         `json:"baseAsset"`
	BaseAssetPrecision         int            `json:"baseAssetPrecision"`
	QuoteAsset                 string         `json:"quoteAsset"`
	QuotePrecision             int            `json:"quotePrecision"`
	QuoteAssetPrecision        int            `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    int            `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   int            `json:"quoteCommissionPrecision"`
	QuoteOrderQtyMarketAllowed bool           `json:"quoteOrderQtyMarketAllowed"`
	IsSpotTradingAllowed       bool           `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool           `json:"isMarginTradingAllowed"`
	OrderTypes                 []string       `json:"orderTypes"`
	Permissions                []string       `json:"permissions"`
	PermissionSets             [][]string     `json:"permissionSets"`
	Filters                    []SymbolFilter `json:"filters"`
	MaxQuoteAmount             Decimal        `json:"maxQuoteAmount"`
	TradeSideType              int            `json:"tradeSideType"`
}
type ExchangeInfo struct {
	Timezone        string           `json:"timezone"`
	ServerTime      int64            `json:"serverTime"`
	RateLimits      []RateLimit      `json:"rateLimits"`
	ExchangeFilters []ExchangeFilter `json:"exchangeFilters"`
	Symbols         []SymbolInfo     `json:"symbols"`
}

// ExchangeInfo gets Spot exchange and symbol metadata.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) ExchangeInfo(ctx context.Context, p ExchangeInfoParams) (*ExchangeInfo, error) {
	q := url.Values{}
	if p.Symbol != "" {
		if err := validateSymbol(p.Symbol); err != nil {
			return nil, fmt.Errorf("failed to get spot exchange information: %w", err)
		}
		q.Set("symbol", p.Symbol)
	}
	if len(p.Symbols) > 0 {
		for _, s := range p.Symbols {
			if err := validateSymbol(s); err != nil {
				return nil, fmt.Errorf("failed to get spot exchange information: %w", err)
			}
		}
		b, _ := json.Marshal(p.Symbols)
		q.Set("symbols", string(b))
	}
	if p.Status != "" {
		q.Set("status", p.Status)
	}
	if p.TradeSideType != "" {
		q.Set("tradeSideType", p.TradeSideType)
	}
	var out ExchangeInfo
	if err := c.get(ctx, "/api/v3/exchangeInfo", q, &out); err != nil {
		return nil, fmt.Errorf("failed to get spot exchange information: %w", err)
	}
	return &out, nil
}

type DepthParams struct {
	Symbol string
	Limit  int
}
type Level [2]Decimal
type Depth struct {
	LastUpdateID json.Number `json:"lastUpdateId"`
	Bids         []Level     `json:"bids"`
	Asks         []Level     `json:"asks"`
}

// Depth gets a Spot order-book snapshot.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Depth(ctx context.Context, p DepthParams) (*Depth, error) {
	if err := validateSymbol(p.Symbol); err != nil {
		return nil, fmt.Errorf("failed to get spot depth: %w", err)
	}
	if p.Limit < 0 || p.Limit > 5000 {
		return nil, fmt.Errorf("failed to get spot depth: limit=out_of_range")
	}
	q := url.Values{"symbol": {p.Symbol}}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	var out Depth
	if err := c.get(ctx, "/api/v3/depth", q, &out); err != nil {
		return nil, fmt.Errorf("failed to get spot depth: %w", err)
	}
	return &out, nil
}

type TradesParams struct {
	Symbol string
	Limit  int
}
type Trade struct {
	ID           *json.Number `json:"id"`
	Price        Decimal      `json:"price"`
	Qty          Decimal      `json:"qty"`
	QuoteQty     Decimal      `json:"quoteQty"`
	Time         int64        `json:"time"`
	IsBuyerMaker bool         `json:"isBuyerMaker"`
	IsBestMatch  bool         `json:"isBestMatch"`
}

// Trades gets recent Spot trades.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Trades(ctx context.Context, p TradesParams) ([]Trade, error) {
	if err := validateSymbol(p.Symbol); err != nil {
		return nil, fmt.Errorf("failed to get spot trades: %w", err)
	}
	if p.Limit < 0 || p.Limit > 1000 {
		return nil, fmt.Errorf("failed to get spot trades: limit=out_of_range")
	}
	q := url.Values{"symbol": {p.Symbol}}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	var out []Trade
	if err := c.get(ctx, "/api/v3/trades", q, &out); err != nil {
		return nil, fmt.Errorf("failed to get spot trades: %w", err)
	}
	return out, nil
}

type BookTicker struct {
	Symbol   string  `json:"symbol"`
	BidPrice Decimal `json:"bidPrice"`
	BidQty   Decimal `json:"bidQty"`
	AskPrice Decimal `json:"askPrice"`
	AskQty   Decimal `json:"askQty"`
}
type BookTickerResult struct {
	One *BookTicker
	All []BookTicker
}

// BookTicker gets one or all Spot best quotes.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) BookTicker(ctx context.Context, symbol string) (*BookTickerResult, error) {
	r, err := decodeOneOrAll[BookTicker](ctx, c, "/api/v3/ticker/bookTicker", symbol, "failed to get spot book ticker")
	if err != nil {
		return nil, err
	}
	return &BookTickerResult{One: r.One, All: r.All}, nil
}

type Ticker24h struct {
	Symbol             string  `json:"symbol"`
	PriceChange        Decimal `json:"priceChange"`
	PriceChangePercent Decimal `json:"priceChangePercent"`
	PrevClosePrice     Decimal `json:"prevClosePrice"`
	LastPrice          Decimal `json:"lastPrice"`
	LastQty            Decimal `json:"lastQty"`
	BidPrice           Decimal `json:"bidPrice"`
	BidQty             Decimal `json:"bidQty"`
	AskPrice           Decimal `json:"askPrice"`
	AskQty             Decimal `json:"askQty"`
	OpenPrice          Decimal `json:"openPrice"`
	HighPrice          Decimal `json:"highPrice"`
	LowPrice           Decimal `json:"lowPrice"`
	Volume             Decimal `json:"volume"`
	QuoteVolume        Decimal `json:"quoteVolume"`
	OpenTime           int64   `json:"openTime"`
	CloseTime          int64   `json:"closeTime"`
	Count              int64   `json:"count"`
}
type Ticker24hResult struct {
	One *Ticker24h
	All []Ticker24h
}

// Ticker24h gets one or all Spot 24-hour tickers.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Ticker24h(ctx context.Context, symbol string) (*Ticker24hResult, error) {
	r, e := decodeOneOrAll[Ticker24h](ctx, c, "/api/v3/ticker/24hr", symbol, "failed to get spot 24-hour ticker")
	if e != nil {
		return nil, e
	}
	return &Ticker24hResult{One: r.One, All: r.All}, nil
}

type PriceTicker struct {
	Symbol string  `json:"symbol"`
	Price  Decimal `json:"price"`
}
type PriceTickerResult struct {
	One *PriceTicker
	All []PriceTicker
}

// PriceTicker gets one or all Spot symbol prices.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) PriceTicker(ctx context.Context, symbol string) (*PriceTickerResult, error) {
	r, e := decodeOneOrAll[PriceTicker](ctx, c, "/api/v3/ticker/price", symbol, "failed to get spot price ticker")
	if e != nil {
		return nil, e
	}
	return &PriceTickerResult{One: r.One, All: r.All}, nil
}

type oneOrAll[T any] struct {
	One *T
	All []T
}

func decodeOneOrAll[T any](ctx context.Context, c *Client, path, symbol, operation string) (*oneOrAll[T], error) {
	q := url.Values{}
	if symbol != "" {
		if err := validateSymbol(symbol); err != nil {
			return nil, fmt.Errorf("%s: %w", operation, err)
		}
		q.Set("symbol", symbol)
	}
	body, err := c.executor.Do(ctx, transport.Request{Method: http.MethodGet, Path: path, Query: q})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	r := &oneOrAll[T]{}
	trim := strings.TrimSpace(string(body))
	if strings.HasPrefix(trim, "[") {
		if err := decode(body, &r.All); err != nil {
			return nil, fmt.Errorf("%s: %w", operation, err)
		}
	} else {
		var one T
		if err := decode(body, &one); err != nil {
			return nil, fmt.Errorf("%s: %w", operation, err)
		}
		r.One = &one
	}
	return r, nil
}
func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	body, err := c.executor.Do(ctx, transport.Request{Method: http.MethodGet, Path: path, Query: q})
	if err != nil {
		return err
	}
	return decode(body, out)
}
func decode(body []byte, out any) error {
	d := json.NewDecoder(strings.NewReader(string(body)))
	d.UseNumber()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}
func validateSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("failed to validate spot symbol: symbol=empty")
	}
	if !spotSymbolPattern.MatchString(symbol) {
		return fmt.Errorf("failed to validate spot symbol: symbol=invalid")
	}
	return nil
}
