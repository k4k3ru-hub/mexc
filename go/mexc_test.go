package mexc

import (
	contractprotocol "github.com/k4k3ru-hub/mexc/go/websocket/contract/protocol"
	spotprotocol "github.com/k4k3ru-hub/mexc/go/websocket/spot/protocol"
	"testing"
)

type spotHandler struct{}

func (spotHandler) HandleControl(*spotprotocol.ControlResponse) {}
func (spotHandler) HandleMarketData(*spotprotocol.Message)      {}

type contractHandler struct{}

func (contractHandler) HandleMessage(*contractprotocol.Message) {}
func TestCompositionRoots(t *testing.T) {
	if c, e := NewSpotRESTClient(nil); e != nil || c == nil {
		t.Fatalf("spot rest: %v", e)
	}
	if c, e := NewContractRESTClient(nil); e != nil || c == nil {
		t.Fatalf("contract rest: %v", e)
	}
	if c, e := NewSpotWebSocketClient(spotHandler{}, nil); e != nil || c == nil {
		t.Fatalf("spot ws: %v", e)
	}
	if c, e := NewContractWebSocketClient(contractHandler{}, nil); e != nil || c == nil {
		t.Fatalf("contract ws: %v", e)
	}
}
