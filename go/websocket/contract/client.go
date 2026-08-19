package contract

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

	"github.com/k4k3ru-hub/mexc/go/websocket/contract/protocol"
	"github.com/k4k3ru-hub/mexc/go/websocket/transport"
)

const DefaultEndpoint = "wss://contract.mexc.com/edge"

var symbolPattern = regexp.MustCompile(`^[A-Z0-9]+_[A-Z0-9]+$`)

type Channel string

const ChannelTicker Channel = "ticker"
const ChannelTrades Channel = "deal"
const ChannelDepth Channel = "depth"
const ChannelFundingRate Channel = "funding.rate"
const ChannelIndexPrice Channel = "index.price"
const ChannelFairPrice Channel = "fair.price"

// Handler receives decoded Contract messages. Pong is consumed by the client.
type Handler interface{ HandleMessage(*protocol.Message) }

// ClientOption configures the Contract WebSocket client.
type ClientOption struct {
	Endpoint          string
	Header            http.Header
	HeartbeatInterval time.Duration
	ReconnectDelay    time.Duration
	Compress          bool
	Dialer            transport.Dialer
	Scheduler         transport.Scheduler
}

// Client is a reconnecting Contract public WebSocket client.
type Client struct {
	option        ClientOption
	handler       Handler
	mu            sync.RWMutex
	subscriptions map[string][]byte
	connMu        sync.Mutex
	conn          transport.Connection
}

// DefaultClientOption returns defaults. Compress is true because MEXC incremental depth merge defaults to enabled.
//
// Version:
//   - 2026-08-19: Added.
func DefaultClientOption() *ClientOption {
	return &ClientOption{Endpoint: DefaultEndpoint, HeartbeatInterval: 20 * time.Second, ReconnectDelay: time.Second, Compress: true, Dialer: transport.GorillaDialer{}, Scheduler: transport.RealScheduler{}}
}

// NewClient creates a Contract WebSocket client without connecting.
//
// Version:
//   - 2026-08-19: Added.
func NewClient(handler Handler, option *ClientOption) (*Client, error) {
	if handler == nil {
		return nil, fmt.Errorf("failed to create contract websocket client: handler=null")
	}
	o := *DefaultClientOption()
	if option != nil {
		o = *option
		if option.Header != nil {
			o.Header = option.Header.Clone()
		}
		if o.Endpoint == "" {
			return nil, fmt.Errorf("failed to create contract websocket client: endpoint=empty")
		}
		if o.HeartbeatInterval == 0 {
			o.HeartbeatInterval = 20 * time.Second
		}
		if o.ReconnectDelay == 0 {
			o.ReconnectDelay = time.Second
		}
		if o.Dialer == nil {
			return nil, fmt.Errorf("failed to create contract websocket client: dialer=null")
		}
		if o.Scheduler == nil {
			o.Scheduler = transport.RealScheduler{}
		}
	}
	if o.HeartbeatInterval < 0 {
		return nil, fmt.Errorf("failed to create contract websocket client: heartbeat_interval=out_of_range")
	}
	if _, err := url.ParseRequestURI(o.Endpoint); err != nil {
		return nil, fmt.Errorf("failed to create contract websocket client: endpoint=invalid: %w", err)
	}
	return &Client{option: o, handler: handler, subscriptions: map[string][]byte{}}, nil
}

// SubscribeTicker registers a Contract ticker subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeTicker(ctx context.Context, symbol string) error {
	return c.subscribe(ctx, ChannelTicker, symbol)
}

// SubscribeTrades registers a Contract public-deal subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeTrades(ctx context.Context, symbol string) error {
	return c.subscribe(ctx, ChannelTrades, symbol)
}

// SubscribeDepth registers Contract incremental depth; compress follows ClientOption and defaults true.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeDepth(ctx context.Context, symbol string) error {
	return c.subscribe(ctx, ChannelDepth, symbol)
}

// SubscribeFundingRate registers a Contract funding-rate subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeFundingRate(ctx context.Context, symbol string) error {
	return c.subscribe(ctx, ChannelFundingRate, symbol)
}

// SubscribeIndexPrice registers a Contract index-price subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeIndexPrice(ctx context.Context, symbol string) error {
	return c.subscribe(ctx, ChannelIndexPrice, symbol)
}

// SubscribeFairPrice registers a Contract fair-price subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) SubscribeFairPrice(ctx context.Context, symbol string) error {
	return c.subscribe(ctx, ChannelFairPrice, symbol)
}

// Unsubscribe removes and sends a Contract subscription.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Unsubscribe(ctx context.Context, ch Channel, symbol string) error {
	key := subscriptionKey(ch, symbol)
	c.mu.Lock()
	_, ok := c.subscriptions[key]
	delete(c.subscriptions, key)
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("failed to unsubscribe contract websocket channel: subscription=not_found")
	}
	req := protocol.Request{Method: "unsub." + string(ch), Param: map[string]any{"symbol": symbol}}
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe contract websocket channel: %w", err)
	}
	return c.write(ctx, b)
}

// Run connects, restores subscriptions, and blocks until context cancellation.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Run(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("failed to run contract websocket client: client=null")
	}
	for {
		if ctx.Err() != nil {
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

func (c *Client) subscribe(ctx context.Context, ch Channel, symbol string) error {
	if symbol == "" {
		return fmt.Errorf("failed to subscribe contract websocket channel: symbol=empty")
	}
	if !symbolPattern.MatchString(symbol) {
		return fmt.Errorf("failed to subscribe contract websocket channel: symbol=invalid")
	}
	param := map[string]any{"symbol": symbol}
	if ch == ChannelDepth {
		param["compress"] = c.option.Compress
	}
	req := protocol.Request{Method: "sub." + string(ch), Param: param}
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to subscribe contract websocket channel: %w", err)
	}
	c.mu.Lock()
	c.subscriptions[subscriptionKey(ch, symbol)] = b
	c.mu.Unlock()
	return c.write(ctx, b)
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
					b, _ := json.Marshal(protocol.Request{Method: "ping"})
					_ = c.write(ctx, b)
				}
			}
		}()
	}
	defer close(done)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		m, err := protocol.Decode(data)
		if err != nil {
			return err
		}
		if m.Pong {
			continue
		}
		c.handler.HandleMessage(m)
	}
}
func (c *Client) restore(ctx context.Context) error {
	c.mu.RLock()
	keys := make([]string, 0, len(c.subscriptions))
	for k := range c.subscriptions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	p := make([][]byte, len(keys))
	for i, k := range keys {
		p[i] = append([]byte(nil), c.subscriptions[k]...)
	}
	c.mu.RUnlock()
	for _, b := range p {
		if err := c.write(ctx, b); err != nil {
			return err
		}
	}
	return nil
}
func (c *Client) write(ctx context.Context, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("failed to write contract websocket message: %w", err)
	}
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn == nil {
		return nil
	}
	if err := c.conn.WriteMessage(transport.TextMessage, payload); err != nil {
		return fmt.Errorf("failed to write contract websocket message: %w", err)
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
func subscriptionKey(ch Channel, symbol string) string {
	return "contract|" + string(ch) + "||" + symbol + "|0"
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
