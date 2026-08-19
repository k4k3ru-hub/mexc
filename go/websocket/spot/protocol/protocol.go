package protocol

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/k4k3ru-hub/mexc/go/websocket/spot/protocol/pb"
	"google.golang.org/protobuf/proto"
)

const MethodSubscription = "SUBSCRIPTION"
const MethodUnsubscription = "UNSUBSCRIPTION"
const MethodPing = "PING"

type Request struct {
	Method string   `json:"method"`
	Params []string `json:"params,omitempty"`
	ID     uint64   `json:"id,omitempty"`
}
type ControlResponse struct {
	ID   uint64 `json:"id"`
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}
type Level struct {
	Price    string
	Quantity string
}
type BookTicker struct {
	Channel             string
	Symbol              string
	SendTime            int64
	BidPrice            string
	BidQuantity         string
	AskPrice            string
	AskQuantity         string
	Version             string
	LastOrderCreateTime int64
}
type Trade struct {
	Price     string
	Quantity  string
	TradeType int32
	Time      int64
	TradeID   string
}
type Trades struct {
	Channel   string
	EventType string
	Symbol    string
	SendTime  int64
	Trades    []Trade
}
type DiffDepth struct {
	Channel             string
	EventType           string
	Symbol              string
	SendTime            int64
	Asks                []Level
	Bids                []Level
	FromVersion         string
	ToVersion           string
	LastOrderCreateTime int64
}
type PartialDepth struct {
	Channel             string
	EventType           string
	Symbol              string
	SendTime            int64
	Asks                []Level
	Bids                []Level
	Version             string
	LastOrderCreateTime int64
}
type Message struct {
	BookTicker   *BookTicker
	Trades       *Trades
	DiffDepth    *DiffDepth
	PartialDepth *PartialDepth
}

// DecodeControl decodes a Spot text control response.
//
// Version:
//   - 2026-08-19: Added.
func DecodeControl(data []byte) (*ControlResponse, error) {
	var out ControlResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("failed to decode spot websocket control response: %w", err)
	}
	return &out, nil
}

// DecodeBinary decodes an official MEXC Protocol Buffers market-data frame.
//
// Version:
//   - 2026-08-19: Added.
func DecodeBinary(data []byte) (*Message, error) {
	var w pb.PushDataV3ApiWrapper
	if err := proto.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("failed to decode spot websocket binary frame: %w", err)
	}
	out := &Message{}
	switch b := w.Body.(type) {
	case *pb.PushDataV3ApiWrapper_PublicAggreBookTicker:
		v := b.PublicAggreBookTicker
		out.BookTicker = &BookTicker{Channel: w.Channel, Symbol: w.GetSymbol(), SendTime: w.GetSendTime(), BidPrice: v.BidPrice, BidQuantity: v.BidQuantity, AskPrice: v.AskPrice, AskQuantity: v.AskQuantity, Version: v.Version, LastOrderCreateTime: v.LastOrderCreateTime}
	case *pb.PushDataV3ApiWrapper_PublicAggreDeals:
		v := b.PublicAggreDeals
		t := make([]Trade, len(v.Deals))
		for i, d := range v.Deals {
			t[i] = Trade{Price: d.Price, Quantity: d.Quantity, TradeType: d.TradeType, Time: d.Time, TradeID: d.TradeId}
		}
		out.Trades = &Trades{Channel: w.Channel, EventType: v.EventType, Symbol: w.GetSymbol(), SendTime: w.GetSendTime(), Trades: t}
	case *pb.PushDataV3ApiWrapper_PublicAggreDepths:
		v := b.PublicAggreDepths
		out.DiffDepth = &DiffDepth{Channel: w.Channel, EventType: v.EventType, Symbol: w.GetSymbol(), SendTime: w.GetSendTime(), Asks: aggreLevels(v.Asks), Bids: aggreLevels(v.Bids), FromVersion: v.FromVersion, ToVersion: v.ToVersion, LastOrderCreateTime: v.LastOrderCreateTime}
	case *pb.PushDataV3ApiWrapper_PublicLimitDepths:
		v := b.PublicLimitDepths
		out.PartialDepth = &PartialDepth{Channel: w.Channel, EventType: v.EventType, Symbol: w.GetSymbol(), SendTime: w.GetSendTime(), Asks: limitLevels(v.Asks), Bids: limitLevels(v.Bids), Version: v.Version, LastOrderCreateTime: v.LastOrderCreateTime}
	default:
		return nil, fmt.Errorf("failed to decode spot websocket binary frame: channel=unsupported")
	}
	return out, nil
}

// IsContinuous reports whether a diff begins immediately after a prior version.
//
// Version:
//   - 2026-08-19: Added.
func (d DiffDepth) IsContinuous(previous string) (bool, error) {
	p, err := strconv.ParseUint(previous, 10, 64)
	if err != nil {
		return false, fmt.Errorf("failed to check spot depth continuity: previous_version=invalid: %w", err)
	}
	from, err := strconv.ParseUint(d.FromVersion, 10, 64)
	if err != nil {
		return false, fmt.Errorf("failed to check spot depth continuity: from_version=invalid: %w", err)
	}
	return from == p+1, nil
}
func aggreLevels(in []*pb.PublicAggreDepthV3ApiItem) []Level {
	out := make([]Level, len(in))
	for i, v := range in {
		out[i] = Level{Price: v.Price, Quantity: v.Quantity}
	}
	return out
}
func limitLevels(in []*pb.PublicLimitDepthV3ApiItem) []Level {
	out := make([]Level, len(in))
	for i, v := range in {
		out[i] = Level{Price: v.Price, Quantity: v.Quantity}
	}
	return out
}

// SubscriptionKey builds a stable key containing market, channel, speed, symbol, and depth level.
//
// Version:
//   - 2026-08-19: Added.
func SubscriptionKey(channel, speed, symbol string, level int) string {
	return strings.Join([]string{"spot", channel, speed, symbol, strconv.Itoa(level)}, "|")
}
