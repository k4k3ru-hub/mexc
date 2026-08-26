package contract

import (
	"context"
	"errors"
	"github.com/k4k3ru-hub/mexc/go/rest/transport"
	"testing"
)

type fakeExecutor struct {
	response []byte
	err      error
	request  transport.Request
}

func (f *fakeExecutor) Do(_ context.Context, r transport.Request) ([]byte, error) {
	f.request = r
	return f.response, f.err
}
func TestDetailAndUSDTPerpetual(t *testing.T) {
	f := &fakeExecutor{response: []byte(`{"success":true,"code":0,"data":[{"symbol":"BTC_USDT","settleCoin":"USDT","contractType":1,"contractSize":0.000000000000000001,"state":0}]}`)}
	c, _ := NewClientWithExecutor(f)
	out, err := c.Detail(context.Background(), "BTC_USDT")
	if err != nil {
		t.Fatal(err)
	}
	if !out[0].IsUSDTMarginedPerpetual() || out[0].ContractSize != "0.000000000000000001" {
		t.Fatalf("unexpected: %#v", out[0])
	}
}
func TestContractError(t *testing.T) {
	f := &fakeExecutor{response: []byte(`{"success":false,"code":6005,"message":"not available","data":null}`)}
	c, _ := NewClientWithExecutor(f)
	_, err := c.Depth(context.Background(), "BTC_USDT")
	var target *ResponseError
	if !errors.As(err, &target) || target.Code.String() != "6005" {
		t.Fatalf("unexpected error: %v", err)
	}
}
func TestFundingHistoryQuery(t *testing.T) {
	f := &fakeExecutor{response: []byte(`{"success":true,"code":0,"data":{"pageSize":10,"totalCount":1,"totalPage":1,"currentPage":2,"resultList":[]}}`)}
	c, _ := NewClientWithExecutor(f)
	_, err := c.FundingRateHistory(context.Background(), FundingRateHistoryParams{Symbol: "BTC_USDT", PageNum: 2, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if f.request.Query.Get("page_num") != "2" || f.request.Query.Get("page_size") != "10" {
		t.Fatalf("query: %v", f.request.Query)
	}
}

// TestFundingRateDecodesCompletePayload verifies current funding and next settlement values.
//
// Version:
//   - 2026-08-26: Added.
func TestFundingRateDecodesCompletePayload(t *testing.T) {
	f := &fakeExecutor{response: []byte(`{"success":true,"code":0,"data":{"symbol":"BTC_USDT","fundingRate":-0.000489,"maxFundingRate":"0.0010","minFundingRate":-0.001,"collectCycle":8,"nextSettleTime":1609833600000,"timestamp":1609829807577}}`)}
	c, _ := NewClientWithExecutor(f)
	out, err := c.FundingRate(context.Background(), "BTC_USDT")
	if err != nil {
		t.Fatalf("FundingRate() error = %v", err)
	}
	if f.request.Path != "/api/v1/contract/funding_rate/BTC_USDT" || out.Symbol != "BTC_USDT" || out.FundingRate != "-0.000489" || out.MaxFundingRate != "0.0010" || out.MinFundingRate != "-0.001" || out.CollectCycle != 8 || out.NextSettleTime != 1609833600000 || out.Timestamp != 1609829807577 {
		t.Fatalf("funding rate = %#v request = %#v", out, f.request)
	}
}

// TestFundingRateHistoryDecodesCompletePayload verifies settled funding pagination and records.
//
// Version:
//   - 2026-08-26: Added.
func TestFundingRateHistoryDecodesCompletePayload(t *testing.T) {
	f := &fakeExecutor{response: []byte(`{"success":true,"code":0,"data":{"pageSize":2,"totalCount":21,"totalPage":11,"currentPage":1,"resultList":[{"symbol":"BTC_USDT","fundingRate":0.000266,"settleTime":1609804800000},{"symbol":"BTC_USDT","fundingRate":"0.000290","settleTime":1609776000000}]}}`)}
	c, _ := NewClientWithExecutor(f)
	out, err := c.FundingRateHistory(context.Background(), FundingRateHistoryParams{Symbol: "BTC_USDT", PageNum: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("FundingRateHistory() error = %v", err)
	}
	if out.PageSize != 2 || out.TotalCount != 21 || out.TotalPage != 11 || out.CurrentPage != 1 || len(out.ResultList) != 2 || out.ResultList[0].FundingRate != "0.000266" || out.ResultList[0].SettleTime != 1609804800000 || out.ResultList[1].FundingRate != "0.000290" {
		t.Fatalf("funding history = %#v", out)
	}
}

// TestIndexAndFairPriceDecodeCompletePayloads verifies current index and fair prices.
//
// Version:
//   - 2026-08-26: Added.
func TestIndexAndFairPriceDecodeCompletePayloads(t *testing.T) {
	f := &fakeExecutor{response: []byte(`{"success":true,"code":0,"data":{"symbol":"BTC_USDT","indexPrice":6861.600,"timestamp":1587442022003}}`)}
	c, _ := NewClientWithExecutor(f)
	index, err := c.IndexPrice(context.Background(), "BTC_USDT")
	if err != nil {
		t.Fatalf("IndexPrice() error = %v", err)
	}
	if index.Symbol != "BTC_USDT" || index.IndexPrice != "6861.600" || index.Timestamp != 1587442022003 {
		t.Fatalf("index price = %#v", index)
	}
	f.response = []byte(`{"success":true,"code":0,"data":{"symbol":"BTC_USDT","fairPrice":"6867.400","timestamp":1587442022004}}`)
	fair, err := c.FairPrice(context.Background(), "BTC_USDT")
	if err != nil {
		t.Fatalf("FairPrice() error = %v", err)
	}
	if fair.Symbol != "BTC_USDT" || fair.FairPrice != "6867.400" || fair.Timestamp != 1587442022004 {
		t.Fatalf("fair price = %#v", fair)
	}
}

func TestDirectDepthAndEnvelopeCommits(t *testing.T) {
	f := &fakeExecutor{response: []byte(`{"asks":[[3968.5,121,4]],"bids":[],"version":9007199254740993,"timestamp":1}`)}
	c, _ := NewClientWithExecutor(f)
	depth, err := c.Depth(context.Background(), "BTC_USDT")
	if err != nil || depth.Version.String() != "9007199254740993" || depth.Asks[0].Price != "3968.5" {
		t.Fatalf("depth: %#v %v", depth, err)
	}
	f.response = []byte(`{"success":true,"code":0,"data":[{"asks":[],"bids":[],"version":2}]}`)
	commits, err := c.DepthCommits(context.Background(), "BTC_USDT", 20)
	if err != nil || len(commits) != 1 || commits[0].Version.String() != "2" {
		t.Fatalf("commits: %#v %v", commits, err)
	}
}

func TestTickerUsesDocumentedQuery(t *testing.T) {
	f := &fakeExecutor{response: []byte(`{"success":true,"code":0,"data":{"symbol":"BTC_USDT","lastPrice":1}}`)}
	c, _ := NewClientWithExecutor(f)
	if _, err := c.Ticker(context.Background(), "BTC_USDT"); err != nil {
		t.Fatal(err)
	}
	if f.request.Path != "/api/v1/contract/ticker" || f.request.Query.Get("symbol") != "BTC_USDT" {
		t.Fatalf("request: %#v", f.request)
	}
}
