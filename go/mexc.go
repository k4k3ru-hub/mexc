// Package mexc exposes the primary composition roots for MEXC public market data.
package mexc

import (
	contractrest "github.com/k4k3ru-hub/mexc/go/rest/contract"
	spotrest "github.com/k4k3ru-hub/mexc/go/rest/spot"
	contractws "github.com/k4k3ru-hub/mexc/go/websocket/contract"
	spotws "github.com/k4k3ru-hub/mexc/go/websocket/spot"
)

type SpotRESTClient = spotrest.Client
type ContractRESTClient = contractrest.Client
type SpotWebSocketClient = spotws.Client
type ContractWebSocketClient = contractws.Client

// NewSpotRESTClient composes a Spot V3 REST client.
//
// Version:
//   - 2026-08-19: Added.
func NewSpotRESTClient(option *RESTClientOption) (*SpotRESTClient, error) {
	return spotrest.NewClient(option)
}

// NewContractRESTClient composes a Contract V1 REST client.
//
// Version:
//   - 2026-08-19: Added.
func NewContractRESTClient(option *RESTClientOption) (*ContractRESTClient, error) {
	return contractrest.NewClient(option)
}

// NewSpotWebSocketClient composes a Spot Protocol Buffers WebSocket client without connecting.
//
// Version:
//   - 2026-08-19: Added.
func NewSpotWebSocketClient(handler spotws.Handler, option *SpotWebSocketClientOption) (*SpotWebSocketClient, error) {
	return spotws.NewClient(handler, option)
}

// NewContractWebSocketClient composes a Contract JSON WebSocket client without connecting.
//
// Version:
//   - 2026-08-19: Added.
func NewContractWebSocketClient(handler contractws.Handler, option *ContractWebSocketClientOption) (*ContractWebSocketClient, error) {
	return contractws.NewClient(handler, option)
}
