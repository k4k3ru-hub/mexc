package mexc

import (
	"github.com/k4k3ru-hub/mexc/go/rest"
	contractws "github.com/k4k3ru-hub/mexc/go/websocket/contract"
	spotws "github.com/k4k3ru-hub/mexc/go/websocket/spot"
)

type RESTClientOption = rest.ClientOption
type SpotWebSocketClientOption = spotws.ClientOption
type ContractWebSocketClientOption = contractws.ClientOption
