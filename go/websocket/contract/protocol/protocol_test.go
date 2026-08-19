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
