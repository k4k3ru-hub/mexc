package spot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/k4k3ru-hub/mexc/go/websocket/spot/protocol"
	"github.com/k4k3ru-hub/mexc/go/websocket/transport"
)

const DefaultEndpoint = "wss://wbs-api.mexc.com/ws"

type Speed string

const Speed100ms Speed = "100ms"
const Speed10ms Speed = "10ms"

type Channel string

const ChannelBookTicker Channel = "aggre.bookTicker"
const ChannelTrades Channel = "aggre.deals"
const ChannelDiffDepth Channel = "aggre.depth"
const ChannelPartialDepth Channel = "limit.depth"

var symbolPattern = regexp.MustCompile(`^[A-Z0-9]+$`)

// Handler receives decoded Spot control and market-data messages.
type Handler interface {
	HandleControl(*protocol.ControlResponse)
	HandleMarketData(*protocol.Message)
}

// ClientOption configures the Spot WebSocket client.
type ClientOption struct {
	Endpoint          string
	Header            http.Header
	HeartbeatInterval time.Duration
	ReconnectDelay    time.Duration
	Dialer            transport.Dialer
	Scheduler         transport.Scheduler
}

// Client is a reconnecting Spot public WebSocket client.
type Client struct {
	option        ClientOption
	handler       Handler
	mu            sync.RWMutex
	subscriptions map[string][]byte
	connMu        sync.Mutex
	conn          transport.Connection
}

// DefaultClientOption returns Spot WebSocket defaults. A zero heartbeat becomes 20 seconds.
//
// Version:
//   - 2026-08-19: Added.
func DefaultClientOption() *ClientOption {
	return &ClientOption{Endpoint: DefaultEndpoint, HeartbeatInterval: 20 * time.Second, ReconnectDelay: time.Second, Dialer: transport.GorillaDialer{}, Scheduler: transport.RealScheduler{}}
}

// NewClient creates a Spot WebSocket client without connecting.
//
// Version:
//   - 2026-08-19: Added.
func NewClient(handler Handler, option *ClientOption) (*Client, error) {
	if handler == nil {
		return nil, fmt.Errorf("failed to create spot websocket client: handler=null")
	}
	o := *DefaultClientOption()
	if option != nil {
		o = *option
		if option.Header != nil {
			o.Header = option.Header.Clone()
		}
		if o.Endpoint == "" {
			return nil, fmt.Errorf("failed to create spot websocket client: endpoint=empty")
		}
		if o.HeartbeatInterval == 0 {
			o.HeartbeatInterval = 20 * time.Second
		}
		if o.ReconnectDelay == 0 {
			o.ReconnectDelay = time.Second
		}
		if o.Dialer == nil {
			return nil, fmt.Errorf("failed to create spot websocket client: dialer=null")
		}
		if o.Scheduler == nil {
			o.Scheduler = transport.RealScheduler{}
		}
	}
	if o.HeartbeatInterval < 0 {
		return nil, fmt.Errorf("failed to create spot websocket client: heartbeat_interval=out_of_range")
	}
	if _, err := url.ParseRequestURI(o.Endpoint); err != nil {
		return nil, fmt.Errorf("failed to create spot websocket client: endpoint=invalid: %w", err)
	}
	return &Client{option: o, handler: handler, subscriptions: map[string][]byte{}}, nil
}

// SubscribeBookTicker registers a Spot book-ticker subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeBookTicker(ctx context.Context, symbol string, speed Speed) error {
	return c.subscribe(ctx, ChannelBookTicker, symbol, speed, 0)
}

// SubscribeTrades registers a Spot aggregate-trades subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeTrades(ctx context.Context, symbol string, speed Speed) error {
	return c.subscribe(ctx, ChannelTrades, symbol, speed, 0)
}

// SubscribeDiffDepth registers a Spot absolute-quantity diff-depth subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeDiffDepth(ctx context.Context, symbol string, speed Speed) error {
	return c.subscribe(ctx, ChannelDiffDepth, symbol, speed, 0)
}

// SubscribePartialDepth registers a Spot partial-depth subscription for level 5, 10, or 20.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribePartialDepth(ctx context.Context, symbol string, level int) error {
	return c.subscribe(ctx, ChannelPartialDepth, symbol, "", level)
}

// Unsubscribe removes and sends a previously registered subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Unsubscribe(ctx context.Context, channel Channel, symbol string, speed Speed, level int) error {
	key := protocol.SubscriptionKey(string(channel), string(speed), symbol, level)
	c.mu.Lock()
	payload, ok := c.subscriptions[key]
	delete(c.subscriptions, key)
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("failed to unsubscribe spot websocket channel: subscription=not_found")
	}
	var req protocol.Request
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("failed to unsubscribe spot websocket channel: %w", err)
	}
	req.Method = protocol.MethodUnsubscription
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe spot websocket channel: %w", err)
	}
	return c.write(ctx, transport.TextMessage, b)
}

// Run connects, restores subscriptions after reconnects, and blocks until cancellation.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Run(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("failed to run spot websocket client: client=null")
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		conn, err := c.option.Dialer.DialContext(ctx, c.option.Endpoint, c.option.Header)
		if err != nil {
			if !wait(ctx, c.option.ReconnectDelay) {
				return nil
			}
			continue
		}
		c.setConn(conn)
		if err := c.restore(ctx); err == nil {
			err = c.session(ctx, conn)
		}
		c.clearConn(conn)
		_ = conn.Close()
		if ctx.Err() != nil {
			return nil
		}
		if !wait(ctx, c.option.ReconnectDelay) {
			return nil
		}
	}
}

func (c *Client) subscribe(ctx context.Context, ch Channel, symbol string, speed Speed, level int) error {
	if !symbolPattern.MatchString(symbol) {
		if symbol == "" {
			return fmt.Errorf("failed to subscribe spot websocket channel: symbol=empty")
		}
		return fmt.Errorf("failed to subscribe spot websocket channel: symbol=invalid")
	}
	var stream string
	if ch == ChannelPartialDepth {
		if level != 5 && level != 10 && level != 20 {
			return fmt.Errorf("failed to subscribe spot websocket channel: depth_level=out_of_range")
		}
		stream = fmt.Sprintf("spot@public.limit.depth.v3.api.pb@%s@%d", symbol, level)
	} else {
		if speed != Speed100ms && speed != Speed10ms {
			return fmt.Errorf("failed to subscribe spot websocket channel: speed=invalid")
		}
		stream = fmt.Sprintf("spot@public.%s.v3.api.pb@%s@%s", ch, speed, symbol)
	}
	req := protocol.Request{Method: protocol.MethodSubscription, Params: []string{stream}}
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to subscribe spot websocket channel: %w", err)
	}
	key := protocol.SubscriptionKey(string(ch), string(speed), symbol, level)
	c.mu.Lock()
	if _, exists := c.subscriptions[key]; !exists && len(c.subscriptions) >= 30 {
		c.mu.Unlock()
		return fmt.Errorf("failed to subscribe spot websocket channel: subscriptions=too_long max_length=30")
	}
	c.subscriptions[key] = b
	c.mu.Unlock()
	return c.write(ctx, transport.TextMessage, b)
}
func (c *Client) session(ctx context.Context, conn transport.Connection) error {
	done := make(chan struct{})
	if c.option.HeartbeatInterval > 0 {
		go func() {
			t := c.option.Scheduler.NewTicker(c.option.HeartbeatInterval)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-done:
					return
				case <-t.C():
					payload, _ := json.Marshal(protocol.Request{Method: protocol.MethodPing})
					_ = c.write(ctx, transport.TextMessage, payload)
				}
			}
		}()
	}
	defer close(done)
	for {
		kind, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if kind == transport.BinaryMessage {
			m, err := protocol.DecodeBinary(data)
			if err != nil {
				return err
			}
			c.handler.HandleMarketData(m)
		} else if kind == transport.TextMessage {
			m, err := protocol.DecodeControl(data)
			if err != nil {
				return err
			}
			c.handler.HandleControl(m)
		}
	}
}
func (c *Client) restore(ctx context.Context) error {
	c.mu.RLock()
	keys := make([]string, 0, len(c.subscriptions))
	for k := range c.subscriptions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	payloads := make([][]byte, len(keys))
	for i, k := range keys {
		payloads[i] = append([]byte(nil), c.subscriptions[k]...)
	}
	c.mu.RUnlock()
	for _, p := range payloads {
		if err := c.write(ctx, transport.TextMessage, p); err != nil {
			return err
		}
	}
	return nil
}
func (c *Client) write(ctx context.Context, kind int, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("failed to write spot websocket message: %w", err)
	}
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn == nil {
		return nil
	}
	if err := c.conn.WriteMessage(kind, payload); err != nil {
		return fmt.Errorf("failed to write spot websocket message: %w", err)
	}
	return nil
}
func (c *Client) setConn(conn transport.Connection) {
	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()
}
func (c *Client) clearConn(conn transport.Connection) {
	c.connMu.Lock()
	if c.conn == conn {
		c.conn = nil
	}
	c.connMu.Unlock()
}
func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
