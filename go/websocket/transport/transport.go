package transport

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Ticker is an injectable heartbeat ticker.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// Scheduler creates heartbeat tickers.
type Scheduler interface{ NewTicker(time.Duration) Ticker }

// RealScheduler uses the process wall clock.
type RealScheduler struct{}

type realTicker struct{ ticker *time.Ticker }

// NewTicker creates a wall-clock ticker.
//
// Version:
//   - 2026-08-19: Added.
func (RealScheduler) NewTicker(interval time.Duration) Ticker {
	return realTicker{ticker: time.NewTicker(interval)}
}

func (t realTicker) C() <-chan time.Time { return t.ticker.C }
func (t realTicker) Stop()               { t.ticker.Stop() }

const TextMessage = websocket.TextMessage
const BinaryMessage = websocket.BinaryMessage

// Connection is the replaceable WebSocket connection boundary.
type Connection interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
}

// Dialer opens WebSocket connections.
type Dialer interface {
	DialContext(context.Context, string, http.Header) (Connection, error)
}

// GorillaDialer adapts gorilla/websocket to Dialer.
type GorillaDialer struct{ Dialer *websocket.Dialer }

// DialContext opens a WebSocket connection.
//
// Version:
//   - 2026-08-19: Added.
func (d GorillaDialer) DialContext(ctx context.Context, endpoint string, h http.Header) (Connection, error) {
	dialer := d.Dialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}
	c, _, err := dialer.DialContext(ctx, endpoint, h)
	return c, err
}
