package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

type Decimal string

// UnmarshalJSON decodes a Contract WebSocket decimal without float conversion.
//
// Version:
//   - 2026-08-19: Added.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("failed to decode contract websocket decimal: destination=null")
	}
	if bytes.Equal(data, []byte("null")) {
		*d = ""
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("failed to decode contract websocket decimal: %w", err)
		}
		*d = Decimal(s)
	} else {
		*d = Decimal(string(data))
	}
	return nil
}

type Request struct {
	Method string         `json:"method"`
	Param  map[string]any `json:"param,omitempty"`
	Gzip   *bool          `json:"gzip,omitempty"`
}
type Envelope struct {
	Channel   string          `json:"channel"`
	Symbol    string          `json:"symbol"`
	Timestamp int64           `json:"ts"`
	Data      json.RawMessage `json:"data"`
	Code      json.Number     `json:"code"`
	Message   string          `json:"message"`
}
type Level struct {
	Price      Decimal
	Volume     Decimal
	OrderCount *json.Number
}

// UnmarshalJSON decodes price, absolute contract volume, and optional order count.
//
// Version:
//   - 2026-08-19: Added.
func (l *Level) UnmarshalJSON(data []byte) error {
	var v []json.RawMessage
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("failed to decode contract websocket depth level: %w", err)
	}
	if len(v) < 2 {
		return fmt.Errorf("failed to decode contract websocket depth level: fields=too_short")
	}
	if err := json.Unmarshal(v[0], &l.Price); err != nil {
		return err
	}
	if err := json.Unmarshal(v[1], &l.Volume); err != nil {
		return err
	}
	if len(v) > 2 && string(v[2]) != "null" {
		var n json.Number
		if err := json.Unmarshal(v[2], &n); err != nil {
			return err
		}
		l.OrderCount = &n
	}
	return nil
}

type Ticker struct {
	Symbol        string      `json:"symbol"`
	LastPrice     Decimal     `json:"lastPrice"`
	Bid1          Decimal     `json:"bid1"`
	Ask1          Decimal     `json:"ask1"`
	Volume24      Decimal     `json:"volume24"`
	HoldVol       Decimal     `json:"holdVol"`
	Lower24Price  Decimal     `json:"lower24Price"`
	High24Price   Decimal     `json:"high24Price"`
	RiseFallRate  Decimal     `json:"riseFallRate"`
	RiseFallValue Decimal     `json:"riseFallValue"`
	IndexPrice    Decimal     `json:"indexPrice"`
	FairPrice     Decimal     `json:"fairPrice"`
	FundingRate   Decimal     `json:"fundingRate"`
	Timestamp     int64       `json:"timestamp"`
	ContractID    json.Number `json:"contractId"`
}
type Trade struct {
	Symbol          string  `json:"symbol"`
	Price           Decimal `json:"p"`
	Volume          Decimal `json:"v"`
	Direction       int     `json:"T"`
	OpenClose       int     `json:"O"`
	AutoTransaction int     `json:"M"`
	TradeTime       int64   `json:"t"`
	MessageTime     int64
}
type Depth struct {
	Symbol       string      `json:"symbol"`
	Asks         []Level     `json:"asks"`
	Bids         []Level     `json:"bids"`
	Version      json.Number `json:"version"`
	BeginVersion json.Number `json:"begin"`
	EndVersion   json.Number `json:"end"`
	Timestamp    int64
}
type FundingRate struct {
	Symbol         string  `json:"symbol"`
	FundingRate    Decimal `json:"fundingRate"`
	NextSettleTime int64   `json:"nextSettleTime"`
	Timestamp      int64
}

// UnmarshalJSON decodes documented funding-rate field variants.
//
// Version:
//   - 2026-08-26: Added support for the documented rate field.
func (v *FundingRate) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("failed to decode contract websocket funding rate: destination=null")
	}
	var wire struct {
		Symbol         string  `json:"symbol"`
		FundingRate    Decimal `json:"fundingRate"`
		Rate           Decimal `json:"rate"`
		NextSettleTime int64   `json:"nextSettleTime"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return fmt.Errorf("failed to decode contract websocket funding rate: %w", err)
	}
	v.Symbol = wire.Symbol
	v.FundingRate = wire.FundingRate
	if v.FundingRate == "" {
		v.FundingRate = wire.Rate
	}
	v.NextSettleTime = wire.NextSettleTime
	return nil
}

type IndexPrice struct {
	Symbol     string  `json:"symbol"`
	IndexPrice Decimal `json:"indexPrice"`
	Timestamp  int64
}

// UnmarshalJSON decodes documented index-price field variants.
//
// Version:
//   - 2026-08-26: Added support for the documented price field.
func (v *IndexPrice) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("failed to decode contract websocket index price: destination=null")
	}
	var wire struct {
		Symbol     string  `json:"symbol"`
		IndexPrice Decimal `json:"indexPrice"`
		Price      Decimal `json:"price"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return fmt.Errorf("failed to decode contract websocket index price: %w", err)
	}
	v.Symbol = wire.Symbol
	v.IndexPrice = wire.IndexPrice
	if v.IndexPrice == "" {
		v.IndexPrice = wire.Price
	}
	return nil
}

type FairPrice struct {
	Symbol    string  `json:"symbol"`
	FairPrice Decimal `json:"fairPrice"`
	Timestamp int64
}

// UnmarshalJSON decodes documented fair-price field variants.
//
// Version:
//   - 2026-08-26: Added support for the documented price field.
func (v *FairPrice) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("failed to decode contract websocket fair price: destination=null")
	}
	var wire struct {
		Symbol    string  `json:"symbol"`
		FairPrice Decimal `json:"fairPrice"`
		Price     Decimal `json:"price"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return fmt.Errorf("failed to decode contract websocket fair price: %w", err)
	}
	v.Symbol = wire.Symbol
	v.FairPrice = wire.FairPrice
	if v.FairPrice == "" {
		v.FairPrice = wire.Price
	}
	return nil
}

type Message struct {
	Envelope    Envelope
	Ticker      *Ticker
	Trades      []Trade
	Depth       *Depth
	FundingRate *FundingRate
	IndexPrice  *IndexPrice
	FairPrice   *FairPrice
	Pong        bool
}

// Decode decodes a Contract JSON control or market-data envelope.
//
// Version:
//   - 2026-08-19: Added.
func Decode(data []byte) (*Message, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var e Envelope
	if err := d.Decode(&e); err != nil {
		return nil, fmt.Errorf("failed to decode contract websocket envelope: %w", err)
	}
	m := &Message{Envelope: e}
	switch e.Channel {
	case "pong":
		m.Pong = true
	case "push.ticker":
		var v Ticker
		if err := decodeData(e.Data, &v); err != nil {
			return nil, err
		}
		m.Ticker = &v
	case "push.deal":
		if err := decodeData(e.Data, &m.Trades); err != nil {
			return nil, err
		}
		for i := range m.Trades {
			m.Trades[i].MessageTime = e.Timestamp
			if m.Trades[i].Symbol == "" {
				m.Trades[i].Symbol = e.Symbol
			}
		}
	case "push.depth", "push.depth.full":
		var v Depth
		if err := decodeData(e.Data, &v); err != nil {
			return nil, err
		}
		v.Timestamp = e.Timestamp
		if v.Symbol == "" {
			v.Symbol = e.Symbol
		}
		m.Depth = &v
	case "push.funding.rate":
		var v FundingRate
		if err := decodeData(e.Data, &v); err != nil {
			return nil, err
		}
		v.Timestamp = e.Timestamp
		if v.Symbol == "" {
			v.Symbol = e.Symbol
		}
		m.FundingRate = &v
	case "push.index.price":
		var v IndexPrice
		if err := decodeData(e.Data, &v); err != nil {
			return nil, err
		}
		v.Timestamp = e.Timestamp
		if v.Symbol == "" {
			v.Symbol = e.Symbol
		}
		m.IndexPrice = &v
	case "push.fair.price":
		var v FairPrice
		if err := decodeData(e.Data, &v); err != nil {
			return nil, err
		}
		v.Timestamp = e.Timestamp
		if v.Symbol == "" {
			v.Symbol = e.Symbol
		}
		m.FairPrice = &v
	}
	return m, nil
}

// IsContinuous reports whether begin/end version fields can follow previous.
//
// Version:
//   - 2026-08-19: Added.
func (d Depth) IsContinuous(previous json.Number) (bool, error) {
	p, err := parse(previous)
	if err != nil {
		return false, fmt.Errorf("failed to check contract depth continuity: previous_version=invalid: %w", err)
	}
	begin, err := parse(d.BeginVersion)
	if err != nil {
		return false, fmt.Errorf("failed to check contract depth continuity: begin_version=invalid: %w", err)
	}
	end, err := parse(d.EndVersion)
	if err != nil {
		return false, fmt.Errorf("failed to check contract depth continuity: end_version=invalid: %w", err)
	}
	return begin <= p+1 && end >= p+1, nil
}
func decodeData(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("failed to decode contract websocket data: %w", err)
	}
	return nil
}
func parse(n json.Number) (uint64, error) { return strconv.ParseUint(n.String(), 10, 64) }
