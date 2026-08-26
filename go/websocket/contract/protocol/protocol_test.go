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
