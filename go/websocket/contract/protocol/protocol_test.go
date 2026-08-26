package protocol

import (
	"encoding/json"
	"testing"
)

func TestDecodeChannelsAndContinuity(t *testing.T) {
	out, err := Decode([]byte(`{"channel":"push.depth","symbol":"BTC_USDT","ts":10,"data":{"asks":[["2","0",3]],"bids":[],"version":12,"begin":11,"end":12}}`))
	if err != nil {
		t.Fatal(err)
	}
	if out.Depth.Asks[0].Volume != "0" || out.Depth.Asks[0].OrderCount.String() != "3" {
		t.Fatalf("level: %#v", out.Depth.Asks[0])
	}
	ok, err := out.Depth.IsContinuous(json.Number("10"))
	if err != nil || !ok {
		t.Fatalf("continuity: %v %v", ok, err)
	}
}
func TestPongIsControl(t *testing.T) {
	out, err := Decode([]byte(`{"channel":"pong","data":1}`))
	if err != nil || !out.Pong {
		t.Fatalf("pong: %#v %v", out, err)
	}
}
func TestDecodeTickerFundingAndPrices(t *testing.T) {
	cases := []string{`{"channel":"push.ticker","data":{"symbol":"BTC_USDT","lastPrice":1.123456789012345678}}`, `{"channel":"push.funding.rate","symbol":"BTC_USDT","ts":1,"data":{"fundingRate":"0.1"}}`, `{"channel":"push.index.price","symbol":"BTC_USDT","ts":1,"data":{"indexPrice":"2"}}`, `{"channel":"push.fair.price","symbol":"BTC_USDT","ts":1,"data":{"fairPrice":"3"}}`}
	for _, in := range cases {
		if _, err := Decode([]byte(in)); err != nil {
			t.Fatal(err)
		}
	}
}

// TestDecodeTickerCompleteOpenInterestPayload verifies Open Interest, prices, and venue time.
//
// Version:
//   - 2026-08-26: Added.
func TestDecodeTickerCompleteOpenInterestPayload(t *testing.T) {
	out, err := Decode([]byte(`{"channel":"push.ticker","symbol":"BTC_USDT","ts":1587442022003,"data":{"ask1":6866.5,"bid1":"6865.0","contractId":9007199254740993,"fairPrice":"6867.400","fundingRate":0.0008,"high24Price":7223.5,"indexPrice":6861.6,"lastPrice":6865.5,"lower24Price":6756,"riseFallRate":-0.0424,"riseFallValue":"-304.5","symbol":"BTC_USDT","timestamp":1587442022004,"holdVol":2284742,"volume24":164586129}}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if out.Ticker == nil {
		t.Fatal("Ticker = nil")
	}
	ticker := out.Ticker
	if ticker.Symbol != "BTC_USDT" || ticker.HoldVol != "2284742" || ticker.FairPrice != "6867.400" || ticker.IndexPrice != "6861.6" || ticker.FundingRate != "0.0008" || ticker.Timestamp != 1587442022004 || ticker.ContractID.String() != "9007199254740993" {
		t.Fatalf("ticker = %#v", ticker)
	}
}

// TestDecodeTickerFallsBackToEnvelopeFields verifies symbol and timestamp fallbacks.
//
// Version:
//   - 2026-08-26: Added.
func TestDecodeTickerFallsBackToEnvelopeFields(t *testing.T) {
	out, err := Decode([]byte(`{"channel":"push.ticker","symbol":"BTC_USDT","ts":1587442022003,"data":{"holdVol":"2284742","fairPrice":"6867.4"}}`))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if out.Ticker == nil || out.Ticker.Symbol != "BTC_USDT" || out.Ticker.Timestamp != 1587442022003 || out.Ticker.HoldVol != "2284742" {
		t.Fatalf("ticker = %#v", out.Ticker)
	}
}

// TestDecodeFundingAndPricesCompletePayloads verifies documented and field-table variants.
//
// Version:
//   - 2026-08-26: Added.
func TestDecodeFundingAndPricesCompletePayloads(t *testing.T) {
	funding, err := Decode([]byte(`{"channel":"push.funding.rate","symbol":"BTC_USDT","ts":1587442022003,"data":{"rate":0.001,"symbol":"BTC_USDT","nextSettleTime":1609833600000}}`))
	if err != nil {
		t.Fatalf("Decode(funding) error = %v", err)
	}
	if funding.FundingRate == nil || funding.FundingRate.Symbol != "BTC_USDT" || funding.FundingRate.FundingRate != "0.001" || funding.FundingRate.NextSettleTime != 1609833600000 || funding.FundingRate.Timestamp != 1587442022003 {
		t.Fatalf("funding = %#v", funding.FundingRate)
	}
	index, err := Decode([]byte(`{"channel":"push.index.price","symbol":"BTC_USDT","ts":1587442022004,"data":{"price":6861.6,"symbol":"BTC_USDT"}}`))
	if err != nil {
		t.Fatalf("Decode(index) error = %v", err)
	}
	if index.IndexPrice == nil || index.IndexPrice.IndexPrice != "6861.6" || index.IndexPrice.Timestamp != 1587442022004 {
		t.Fatalf("index = %#v", index.IndexPrice)
	}
	fair, err := Decode([]byte(`{"channel":"push.fair.price","symbol":"BTC_USDT","ts":1587442022005,"data":{"fairPrice":"6867.400","symbol":"BTC_USDT"}}`))
	if err != nil {
		t.Fatalf("Decode(fair) error = %v", err)
	}
	if fair.FairPrice == nil || fair.FairPrice.FairPrice != "6867.400" || fair.FairPrice.Timestamp != 1587442022005 {
		t.Fatalf("fair = %#v", fair.FairPrice)
	}
}
