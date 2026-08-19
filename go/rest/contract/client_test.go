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
