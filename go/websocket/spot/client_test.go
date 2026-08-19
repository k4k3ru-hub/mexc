package spot

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/k4k3ru-hub/mexc/go/websocket/spot/protocol"
	"github.com/k4k3ru-hub/mexc/go/websocket/transport"
)

type handler struct{}

func (handler) HandleControl(*protocol.ControlResponse) {}
func (handler) HandleMarketData(*protocol.Message)      {}

type fakeConn struct {
	mu     sync.Mutex
	writes [][]byte
	cancel context.CancelFunc
}

func (f *fakeConn) ReadMessage() (int, []byte, error) {
	if f.cancel != nil {
		f.cancel()
	}
	return 0, nil, errors.New("closed")
}
func (f *fakeConn) WriteMessage(_ int, b []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, append([]byte(nil), b...))
	return nil
}
func (f *fakeConn) Close() error { return nil }

type fakeDialer struct {
	mu          sync.Mutex
	count       int
	connections []*fakeConn
	cancel      context.CancelFunc
}

func (d *fakeDialer) DialContext(_ context.Context, _ string, _ http.Header) (transport.Connection, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.count++
	c := &fakeConn{}
	if d.count == 2 {
		c.cancel = d.cancel
	}
	d.connections = append(d.connections, c)
	return c, nil
}

func TestSubscriptionPayloadAndReconnectRestoration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	d := &fakeDialer{cancel: cancel}
	o := DefaultClientOption()
	o.Endpoint = "ws://example.test"
	o.Dialer = d
	o.HeartbeatInterval = time.Hour
	o.ReconnectDelay = time.Millisecond
	c, err := NewClient(handler{}, o)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SubscribeTrades(ctx, "BTCUSDT", Speed10ms); err != nil {
		t.Fatal(err)
	}
	if err := c.SubscribePartialDepth(ctx, "ETHUSDT", 20); err != nil {
		t.Fatal(err)
	}
	if err := c.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if d.count < 2 {
		t.Fatalf("expected reconnect: %d", d.count)
	}
	for i, conn := range d.connections {
		conn.mu.Lock()
		n := len(conn.writes)
		conn.mu.Unlock()
		if n != 2 {
			t.Fatalf("connection %d restored %d subscriptions", i, n)
		}
	}
}

func TestConstructorDoesNotConnectOrMutateOption(t *testing.T) {
	d := &fakeDialer{}
	h := http.Header{"X-Test": {"one"}}
	o := DefaultClientOption()
	o.Endpoint = "ws://example.test"
	o.Dialer = d
	o.Header = h
	c, err := NewClient(handler{}, o)
	if err != nil {
		t.Fatal(err)
	}
	if d.count != 0 {
		t.Fatal("constructor connected")
	}
	h.Set("X-Test", "two")
	if c.option.Header.Get("X-Test") != "one" {
		t.Fatal("option header was not cloned")
	}
}
