package contract

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

// DefaultBaseURL is the Contract V1 domain enabled in January 2026.
const DefaultBaseURL = "https://api.mexc.com"
const DeprecatedBaseURL = "https://contract.mexc.com"

var contractSymbolPattern = regexp.MustCompile(`^[A-Z0-9]+_[A-Z0-9]+$`)

// Decimal preserves a Contract JSON string or number lexeme.
type Decimal string

// UnmarshalJSON decodes a decimal string or number.
//
// Version:
//   - 2026-08-19: Added.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("failed to decode contract decimal: destination=null")
	}
	if string(data) == "null" {
		*d = ""
		return nil
	}
	var s string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("failed to decode contract decimal: %w", err)
		}
	} else {
		s = string(data)
	}
	*d = Decimal(s)
	return nil
}

// Client provides Contract V1 public market-data operations.
type Client struct{ executor transport.Executor }

// NewClient creates a Contract REST client without making a network request.
//
// Version:
//   - 2026-08-19: Added.
func NewClient(option *rest.ClientOption) (*Client, error) {
	executor, err := rest.NewClient(option, DefaultBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract rest client: %w", err)
	}
	return NewClientWithExecutor(executor)
}

// NewClientWithExecutor creates a Contract REST client with an executor.
//
// Version:
//   - 2026-08-19: Added.
func NewClientWithExecutor(executor transport.Executor) (*Client, error) {
	if executor == nil {
		return nil, fmt.Errorf("failed to create contract rest client: executor=null")
	}
	return &Client{executor: executor}, nil
}

// ResponseError represents a Contract success=false envelope.
type ResponseError struct {
	Code    json.Number
	Message string
}

// Error returns the Contract response error.
//
// Version:
//   - 2026-08-19: Added.
func (e *ResponseError) Error() string {
	if e == nil {
		return "failed to decode contract response: response_error=null"
	}
	return fmt.Sprintf("failed to decode contract response: venue rejected request: code=%s message=%q", e.Code.String(), e.Message)
}

type ContractDetail struct {
	Symbol                string      `json:"symbol"`
	DisplayName           string      `json:"displayName"`
	DisplayNameEn         string      `json:"displayNameEn"`
	BaseCoin              string      `json:"baseCoin"`
	QuoteCoin             string      `json:"quoteCoin"`
	SettleCoin            string      `json:"settleCoin"`
	ContractSize          Decimal     `json:"contractSize"`
	ContractType          int         `json:"contractType"`
	State                 int         `json:"state"`
	PriceUnit             Decimal     `json:"priceUnit"`
	VolUnit               Decimal     `json:"volUnit"`
	MinVol                Decimal     `json:"minVol"`
	MaxVol                Decimal     `json:"maxVol"`
	MinLeverage           Decimal     `json:"minLeverage"`
	MaxLeverage           Decimal     `json:"maxLeverage"`
	PriceScale            int         `json:"priceScale"`
	VolScale              int         `json:"volScale"`
	AmountScale           int         `json:"amountScale"`
	MakerFeeRate          Decimal     `json:"makerFeeRate"`
	TakerFeeRate          Decimal     `json:"takerFeeRate"`
	MaintenanceMarginRate Decimal     `json:"maintenanceMarginRate"`
	InitialMarginRate     Decimal     `json:"initialMarginRate"`
	RiskBaseVol           Decimal     `json:"riskBaseVol"`
	RiskIncrVol           Decimal     `json:"riskIncrVol"`
	RiskIncrMmr           Decimal     `json:"riskIncrMmr"`
	RiskIncrImr           Decimal     `json:"riskIncrImr"`
	RiskLevelLimit        int         `json:"riskLevelLimit"`
	IndexOrigin           []string    `json:"indexOrigin"`
	PriceSource           int         `json:"priceSource"`
	FundingRateCycle      int         `json:"fundingRateCycle"`
	MaxFundingRate        Decimal     `json:"maxFundingRate"`
	MinFundingRate        Decimal     `json:"minFundingRate"`
	ApiAllowed            bool        `json:"apiAllowed"`
	ContractID            json.Number `json:"contractId"`
}

// IsUSDTMarginedPerpetual reports whether confirmed wire fields identify a USDT perpetual.
//
// Version:
//   - 2026-08-19: Added.
func (d ContractDetail) IsUSDTMarginedPerpetual() bool {
	return d.SettleCoin == "USDT" && d.ContractType == 1
}

// Detail gets Contract metadata, optionally for one exact symbol.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Detail(ctx context.Context, symbol string) ([]ContractDetail, error) {
	q := url.Values{}
	if symbol != "" {
		if err := validateSymbol(symbol); err != nil {
			return nil, fmt.Errorf("failed to get contract detail: %w", err)
		}
		q.Set("symbol", symbol)
	}
	var out []ContractDetail
	if err := c.get(ctx, "/api/v1/contract/detail", q, &out, false); err != nil {
		return nil, fmt.Errorf("failed to get contract detail: %w", err)
	}
	return out, nil
}

type DepthLevel struct {
	Price      Decimal
	Volume     Decimal
	OrderCount *json.Number
}

// UnmarshalJSON decodes a Contract depth array while preserving optional order count.
//
// Version:
//   - 2026-08-19: Added.
func (l *DepthLevel) UnmarshalJSON(data []byte) error {
	var v []json.RawMessage
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("failed to decode contract depth level: %w", err)
	}
	if len(v) < 2 {
		return fmt.Errorf("failed to decode contract depth level: fields=too_short")
	}
	if err := json.Unmarshal(v[0], &l.Price); err != nil {
		return fmt.Errorf("failed to decode contract depth level: %w", err)
	}
	if err := json.Unmarshal(v[1], &l.Volume); err != nil {
		return fmt.Errorf("failed to decode contract depth level: %w", err)
	}
	if len(v) > 2 && string(v[2]) != "null" {
		var n json.Number
		if err := json.Unmarshal(v[2], &n); err != nil {
			return fmt.Errorf("failed to decode contract depth level: %w", err)
		}
		l.OrderCount = &n
	}
	return nil
}

type Depth struct {
	Symbol    string       `json:"symbol"`
	Asks      []DepthLevel `json:"asks"`
	Bids      []DepthLevel `json:"bids"`
	Version   json.Number  `json:"version"`
	Timestamp int64        `json:"timestamp"`
}

// Depth gets a Contract depth snapshot.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Depth(ctx context.Context, symbol string) (*Depth, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, fmt.Errorf("failed to get contract depth: %w", err)
	}
	body, err := c.executor.Do(ctx, transport.Request{Method: http.MethodGet, Path: "/api/v1/contract/depth/" + symbol})
	if err != nil {
		return nil, fmt.Errorf("failed to get contract depth: %w", err)
	}
	var failure struct {
		Success *bool       `json:"success"`
		Code    json.Number `json:"code"`
		Message string      `json:"message"`
	}
	if err := decode(body, &failure); err == nil && failure.Success != nil && !*failure.Success {
		return nil, fmt.Errorf("failed to get contract depth: %w", &ResponseError{Code: failure.Code, Message: failure.Message})
	}
	var out Depth
	if err := decode(body, &out); err != nil {
		return nil, fmt.Errorf("failed to get contract depth: %w", err)
	}
	out.Symbol = symbol
	return &out, nil
}

// DepthCommits gets a bounded Contract depth snapshot for stream resynchronization.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) DepthCommits(ctx context.Context, symbol string, limit int) ([]Depth, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, fmt.Errorf("failed to get contract depth commits: %w", err)
	}
	if limit <= 0 {
		return nil, fmt.Errorf("failed to get contract depth commits: limit=out_of_range")
	}
	var out []Depth
	if err := c.get(ctx, "/api/v1/contract/depth_commits/"+symbol+"/"+strconv.Itoa(limit), nil, &out, false); err != nil {
		return nil, fmt.Errorf("failed to get contract depth commits: %w", err)
	}
	for i := range out {
		out[i].Symbol = symbol
	}
	return out, nil
}

type Ticker struct {
	Symbol        string      `json:"symbol"`
	LastPrice     Decimal     `json:"lastPrice"`
	Bid1          Decimal     `json:"bid1"`
	Ask1          Decimal     `json:"ask1"`
	Volume24      Decimal     `json:"volume24"`
	Amount24      Decimal     `json:"amount24"`
	High24Price   Decimal     `json:"high24Price"`
	Lower24Price  Decimal     `json:"lower24Price"`
	RiseFallRate  Decimal     `json:"riseFallRate"`
	RiseFallValue Decimal     `json:"riseFallValue"`
	IndexPrice    Decimal     `json:"indexPrice"`
	FairPrice     Decimal     `json:"fairPrice"`
	FundingRate   Decimal     `json:"fundingRate"`
	MaxBidPrice   Decimal     `json:"maxBidPrice"`
	MinAskPrice   Decimal     `json:"minAskPrice"`
	HoldVol       Decimal     `json:"holdVol"`
	Timestamp     int64       `json:"timestamp"`
	ContractID    json.Number `json:"contractId"`
}
type TickerResult struct {
	One *Ticker
	All []Ticker
}

// Ticker gets one or all Contract tickers using the documented paths.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Ticker(ctx context.Context, symbol string) (*TickerResult, error) {
	path := "/api/v1/contract/ticker"
	query := url.Values{}
	var raw json.RawMessage
	if symbol != "" {
		if err := validateSymbol(symbol); err != nil {
			return nil, fmt.Errorf("failed to get contract ticker: %w", err)
		}
		query.Set("symbol", symbol)
	}
	if err := c.get(ctx, path, query, &raw, false); err != nil {
		return nil, fmt.Errorf("failed to get contract ticker: %w", err)
	}
	out := &TickerResult{}
	if len(raw) > 0 && raw[0] == '[' {
		if err := decode(raw, &out.All); err != nil {
			return nil, fmt.Errorf("failed to get contract ticker: %w", err)
		}
	} else {
		var one Ticker
		if err := decode(raw, &one); err != nil {
			return nil, fmt.Errorf("failed to get contract ticker: %w", err)
		}
		out.One = &one
	}
	return out, nil
}

type Deal struct {
	Symbol string  `json:"symbol"`
	Price  Decimal `json:"price"`
	Vol    Decimal `json:"vol"`
	Side   int     `json:"side"`
	T      int64   `json:"t"`
	O      int     `json:"O"`
	M      int     `json:"M"`
}

// Deals gets recent public Contract deals.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Deals(ctx context.Context, symbol string) ([]Deal, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, fmt.Errorf("failed to get contract deals: %w", err)
	}
	var out []Deal
	if err := c.get(ctx, "/api/v1/contract/deals/"+symbol, nil, &out, true); err != nil {
		return nil, fmt.Errorf("failed to get contract deals: %w", err)
	}
	for i := range out {
		out[i].Symbol = symbol
	}
	return out, nil
}

type IndexPrice struct {
	Symbol     string  `json:"symbol"`
	IndexPrice Decimal `json:"indexPrice"`
	Timestamp  int64   `json:"timestamp"`
}
type FairPrice struct {
	Symbol    string  `json:"symbol"`
	FairPrice Decimal `json:"fairPrice"`
	Timestamp int64   `json:"timestamp"`
}

// IndexPrice gets the current Contract index price.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) IndexPrice(ctx context.Context, symbol string) (*IndexPrice, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, fmt.Errorf("failed to get contract index price: %w", err)
	}
	var out IndexPrice
	if err := c.get(ctx, "/api/v1/contract/index_price/"+symbol, nil, &out, false); err != nil {
		return nil, fmt.Errorf("failed to get contract index price: %w", err)
	}
	return &out, nil
}

// FairPrice gets the current Contract fair price without renaming it to mark price.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) FairPrice(ctx context.Context, symbol string) (*FairPrice, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, fmt.Errorf("failed to get contract fair price: %w", err)
	}
	var out FairPrice
	if err := c.get(ctx, "/api/v1/contract/fair_price/"+symbol, nil, &out, false); err != nil {
		return nil, fmt.Errorf("failed to get contract fair price: %w", err)
	}
	return &out, nil
}

type FundingRate struct {
	Symbol         string  `json:"symbol"`
	FundingRate    Decimal `json:"fundingRate"`
	MaxFundingRate Decimal `json:"maxFundingRate"`
	MinFundingRate Decimal `json:"minFundingRate"`
	CollectCycle   int     `json:"collectCycle"`
	NextSettleTime int64   `json:"nextSettleTime"`
	Timestamp      int64   `json:"timestamp"`
}

// FundingRate gets the current public Contract funding rate.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) FundingRate(ctx context.Context, symbol string) (*FundingRate, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, fmt.Errorf("failed to get contract funding rate: %w", err)
	}
	var out FundingRate
	if err := c.get(ctx, "/api/v1/contract/funding_rate/"+symbol, nil, &out, false); err != nil {
		return nil, fmt.Errorf("failed to get contract funding rate: %w", err)
	}
	return &out, nil
}

type FundingRateHistoryParams struct {
	Symbol   string
	PageNum  int
	PageSize int
}
type FundingRateRecord struct {
	Symbol      string  `json:"symbol"`
	FundingRate Decimal `json:"fundingRate"`
	SettleTime  int64   `json:"settleTime"`
}
type FundingRateHistory struct {
	PageSize    int                 `json:"pageSize"`
	TotalCount  int                 `json:"totalCount"`
	TotalPage   int                 `json:"totalPage"`
	CurrentPage int                 `json:"currentPage"`
	ResultList  []FundingRateRecord `json:"resultList"`
}

// FundingRateHistory gets public Contract funding settlement history.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) FundingRateHistory(ctx context.Context, p FundingRateHistoryParams) (*FundingRateHistory, error) {
	if err := validateSymbol(p.Symbol); err != nil {
		return nil, fmt.Errorf("failed to get contract funding rate history: %w", err)
	}
	if p.PageNum < 0 || p.PageSize < 0 || p.PageSize > 1000 {
		return nil, fmt.Errorf("failed to get contract funding rate history: pagination=out_of_range")
	}
	q := url.Values{"symbol": {p.Symbol}}
	if p.PageNum > 0 {
		q.Set("page_num", strconv.Itoa(p.PageNum))
	}
	if p.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(p.PageSize))
	}
	var out FundingRateHistory
	if err := c.get(ctx, "/api/v1/contract/funding_rate/history", q, &out, false); err != nil {
		return nil, fmt.Errorf("failed to get contract funding rate history: %w", err)
	}
	return &out, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any, emptyAllowed bool) error {
	body, err := c.executor.Do(ctx, transport.Request{Method: http.MethodGet, Path: path, Query: q})
	if err != nil {
		return err
	}
	var envelope struct {
		Success bool            `json:"success"`
		Code    json.Number     `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := decode(body, &envelope); err != nil {
		return err
	}
	if !envelope.Success {
		return &ResponseError{Code: envelope.Code, Message: envelope.Message}
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		if emptyAllowed {
			return nil
		}
		return fmt.Errorf("failed to decode contract response: data=null")
	}
	return decode(envelope.Data, out)
}
func decode(data []byte, out any) error {
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.UseNumber()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("failed to decode contract response: %w", err)
	}
	return nil
}
func validateSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("failed to validate contract symbol: symbol=empty")
	}
	if !contractSymbolPattern.MatchString(symbol) {
		return fmt.Errorf("failed to validate contract symbol: symbol=invalid")
	}
	return nil
}
