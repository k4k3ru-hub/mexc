package contract

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/k4k3ru-hub/mexc/go/websocket/contract/protocol"
)

type handler struct{ messages int }

func (h *handler) HandleMessage(*protocol.Message) { h.messages++ }

func TestSubscriptionPayloadsAndKeys(t *testing.T) {
	h := &handler{}
	c, err := NewClient(h, nil)
	if err != nil {
		t.Fatal(err)
	}
	methods := map[Channel]func(context.Context, string) error{ChannelTicker: c.SubscribeTicker, ChannelTrades: c.SubscribeTrades, ChannelDepth: c.SubscribeDepth, ChannelFundingRate: c.SubscribeFundingRate, ChannelIndexPrice: c.SubscribeIndexPrice, ChannelFairPrice: c.SubscribeFairPrice}
	for channel, subscribe := range methods {
		if err := subscribe(context.Background(), "BTC_USDT"); err != nil {
			t.Fatal(err)
		}
		key := subscriptionKey(channel, "BTC_USDT")
		var req protocol.Request
		if err := json.Unmarshal(c.subscriptions[key], &req); err != nil {
			t.Fatal(err)
		}
		if req.Method != "sub."+string(channel) {
			t.Fatalf("channel %s: %s", channel, req.Method)
		}
	}
	var depth protocol.Request
	if err := json.Unmarshal(c.subscriptions[subscriptionKey(ChannelDepth, "BTC_USDT")], &depth); err != nil {
		t.Fatal(err)
	}
	if depth.Param["compress"] != true {
		t.Fatalf("compress default: %#v", depth.Param)
	}
}

func TestValidationAndConstructor(t *testing.T) {
	c, err := NewClient(&handler{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SubscribeTicker(context.Background(), "BTCUSDT"); err == nil {
		t.Fatal("expected symbol format error")
	}
	o := DefaultClientOption()
	o.HeartbeatInterval = -1
	if _, err := NewClient(&handler{}, o); err == nil {
		t.Fatal("expected heartbeat error")
	}
}
