package spot

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/k4k3ru-hub/mexc/go/rest/transport"
)

type fakeExecutor struct {
	mu       sync.Mutex
	requests []transport.Request
	response func(transport.Request) ([]byte, error)
}

func (f *fakeExecutor) Do(_ context.Context, r transport.Request) ([]byte, error) {
	f.mu.Lock()
	f.requests = append(f.requests, r)
	f.mu.Unlock()
	return f.response(r)
}

func TestDepthPreservesPrecisionAndRequest(t *testing.T) {
	f := &fakeExecutor{response: func(r transport.Request) ([]byte, error) {
		if r.Path != "/api/v3/depth" || r.Query.Get("symbol") != "BTCUSDT" || r.Query.Get("limit") != "100" {
			t.Fatalf("unexpected request: %#v", r)
		}
		return []byte(`{"lastUpdateId":9007199254740993,"bids":[["0.123456789012345678","1.000000000000000001"]],"asks":[]}`), nil
	}}
	c, _ := NewClientWithExecutor(f)
	out, err := c.Depth(context.Background(), DepthParams{Symbol: "BTCUSDT", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if out.LastUpdateID.String() != "9007199254740993" || out.Bids[0][0] != "0.123456789012345678" {
		t.Fatalf("precision lost: %#v", out)
	}
}

func TestTradesNullableID(t *testing.T) {
	f := &fakeExecutor{response: func(transport.Request) ([]byte, error) {
		return []byte(`[{"id":null,"price":"1","qty":"2","quoteQty":"2","time":1,"isBuyerMaker":true,"isBestMatch":false}]`), nil
	}}
	c, _ := NewClientWithExecutor(f)
	out, err := c.Trades(context.Background(), TradesParams{Symbol: "BTCUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].ID != nil {
		t.Fatal("expected null ID")
	}
}

func TestOneAndAllTicker(t *testing.T) {
	responses := map[string]string{"BTCUSDT": `{"symbol":"BTCUSDT","price":"1.1"}`, "": `[{"symbol":"BTCUSDT","price":"1.1"}]`}
	f := &fakeExecutor{response: func(r transport.Request) ([]byte, error) { return []byte(responses[r.Query.Get("symbol")]), nil }}
	c, _ := NewClientWithExecutor(f)
	one, err := c.PriceTicker(context.Background(), "BTCUSDT")
	if err != nil || one.One == nil || len(one.All) != 0 {
		t.Fatalf("one: %#v %v", one, err)
	}
	all, err := c.PriceTicker(context.Background(), "")
	if err != nil || all.One != nil || len(all.All) != 1 {
		t.Fatalf("all: %#v %v", all, err)
	}
}

func TestConcurrentParametersDoNotMix(t *testing.T) {
	f := &fakeExecutor{response: func(r transport.Request) ([]byte, error) {
		s := r.Query.Get("symbol")
		return []byte(fmt.Sprintf(`{"lastUpdateId":1,"bids":[["%s","1"]],"asks":[]}`, s)), nil
	}}
	c, _ := NewClientWithExecutor(f)
	var wg sync.WaitGroup
	for _, s := range []string{"BTCUSDT", "ETHUSDT"} {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := c.Depth(context.Background(), DepthParams{Symbol: s})
			if err != nil || string(out.Bids[0][0]) != s {
				t.Errorf("mixed %s: %#v %v", s, out, err)
			}
		}()
	}
	wg.Wait()
}

func TestValidation(t *testing.T) {
	c, _ := NewClientWithExecutor(&fakeExecutor{})
	if _, err := c.Depth(context.Background(), DepthParams{Symbol: "BTC_USDT"}); err == nil {
		t.Fatal("expected symbol error")
	}
	if _, err := c.Trades(context.Background(), TradesParams{Symbol: "btcusdt"}); err == nil {
		t.Fatal("expected lowercase error")
	}
}
