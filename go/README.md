# MEXC Public Market Data SDK for Go

`github.com/k4k3ru-hub/mexc/go` is an independent client module for MEXC Spot V3 and USDT-margined perpetual Contract V1 public market data. It does not contain authentication, trading, account APIs, K4K3RU normalization, storage, or redistribution.

## Supported APIs

Spot REST (`https://api.mexc.com` by default):

- `GET /api/v3/exchangeInfo`
- `GET /api/v3/depth`
- `GET /api/v3/trades`
- `GET /api/v3/ticker/bookTicker`
- `GET /api/v3/ticker/24hr`
- `GET /api/v3/ticker/price`

Contract REST (`https://api.mexc.com` by default):

- `GET /api/v1/contract/detail`
- `GET /api/v1/contract/depth/{symbol}`
- `GET /api/v1/contract/depth_commits/{symbol}/{limit}`
- `GET /api/v1/contract/ticker` with optional `symbol` query
- `GET /api/v1/contract/deals/{symbol}`
- `GET /api/v1/contract/index_price/{symbol}`
- `GET /api/v1/contract/fair_price/{symbol}`
- `GET /api/v1/contract/funding_rate/{symbol}`
- `GET /api/v1/contract/funding_rate/history`

Spot WebSocket (`wss://wbs-api.mexc.com/ws`) decodes the Protocol Buffers channels `aggre.bookTicker`, `aggre.deals`, `aggre.depth`, and `limit.depth` (levels 5, 10, and 20). Contract WebSocket (`wss://contract.mexc.com/edge`) decodes the JSON channels `ticker`, `deal`, `depth`, `funding.rate`, `index.price`, and `fair.price`.

The Spot symbol format is `BTCUSDT`; the Contract format is `BTC_USDT`. The module validates these forms and never inserts or removes the underscore.

## Usage

```go
spotClient, err := mexc.NewSpotRESTClient(nil)
if err != nil { return err }
book, err := spotClient.Depth(ctx, spot.DepthParams{Symbol: "BTCUSDT", Limit: 100})
```

```go
contractClient, err := mexc.NewContractRESTClient(nil)
if err != nil { return err }
detail, err := contractClient.Detail(ctx, "BTC_USDT")
if err != nil { return err }
ticker, err := contractClient.Ticker(ctx, "BTC_USDT")
if err != nil { return err }
// ticker.One.HoldVol, ticker.One.FairPrice, ticker.One.Timestamp
```

Initialize current perpetual funding and price state through REST:

```go
funding, err := contractClient.FundingRate(ctx, "BTC_USDT")
if err != nil { return err }
// funding.FundingRate, funding.NextSettleTime, funding.CollectCycle

fair, err := contractClient.FairPrice(ctx, "BTC_USDT")
if err != nil { return err }
index, err := contractClient.IndexPrice(ctx, "BTC_USDT")
if err != nil { return err }
```

Retrieve settled funding rates for startup backfill or gap reconciliation:

```go
history, err := contractClient.FundingRateHistory(ctx, contract.FundingRateHistoryParams{
	Symbol:   "BTC_USDT",
	PageNum:  1,
	PageSize: 1000,
})
```

Subscribe to current Contract values after initialization:

```go
if err := contractWebSocket.SubscribeTicker(ctx, "BTC_USDT"); err != nil { return err }
if err := contractWebSocket.SubscribeFundingRate(ctx, "BTC_USDT"); err != nil { return err }
if err := contractWebSocket.SubscribeFairPrice(ctx, "BTC_USDT"); err != nil { return err }
if err := contractWebSocket.SubscribeIndexPrice(ctx, "BTC_USDT"); err != nil { return err }
```

MEXC calls its mark-price equivalent `fairPrice`; the SDK preserves that venue-native name. `nextSettleTime` is the next funding timestamp, `collectCycle` is the current funding interval, and ticker `holdVol` is the venue's open-position volume field. Downstream normalization may map these to mark price, next funding time, funding interval, and open interest while retaining the original source semantics.

For Open Interest state, combine REST Contract Detail and Ticker at startup,
then use the WebSocket Ticker for subsequent updates. `holdVol` is preserved as
the venue-reported contract quantity, while `contractSize` and `baseCoin` come
from Contract Detail. Consumers may derive base-asset quantity as
`holdVol * contractSize` and notional value as that quantity multiplied by
`fairPrice`. Use the ticker `timestamp` as the venue observation time;
WebSocket decoding falls back to the envelope `ts` when the nested timestamp
is absent. Quantity normalization and notional calculation belong to the
consuming service rather than this SDK.

## CLI

Build the CLI without initiating a network connection:

```sh
go build -o build/mexc ./cli
```

The initial REST command is Spot Exchange Information:

```sh
./build/mexc rest spot exchange-info
./build/mexc rest spot exchange-info --symbol BTCUSDT
./build/mexc rest spot exchange-info --symbols BTCUSDT,ETHUSDT --status 1 --trade-side-type 2
```

`--base-url` replaces the Spot REST endpoint, including for local fixtures and test servers.

Constructing a WebSocket client does not connect. Register subscriptions, then call `Run(ctx)`. `Run` reconnects and restores registered subscriptions. A reconnect invalidates any caller-owned depth state: obtain a new REST snapshot and apply updates only after the documented version-continuity condition is satisfied. This module intentionally does not claim to provide a complete or lossless order book and contains no shared order-book cache.

REST and WebSocket endpoints are replaceable with `rest.ClientOption.BaseURL` and the respective WebSocket `ClientOption.Endpoint`. HTTP clients, REST executors, WebSocket dialers, and heartbeat schedulers are injectable for offline tests.

## Spot Protocol Buffers

The `.proto` files under `websocket/spot/protocol/proto` are an unmodified snapshot of the [official `mexcdevelop/websocket-proto` repository](https://github.com/mexcdevelop/websocket-proto), commit `7b8ac7a6681f28551612a5a7cefbb7e09b56bb85` (upstream commit dated 2026-06-17; retrieved 2026-08-19). Generated code uses `protoc-gen-go v1.36.10` and is not edited manually.

Regenerate it with `protoc` and `protoc-gen-go v1.36.10` on `PATH`:

```sh
go generate ./websocket/spot/protocol/pb
git diff --exit-code -- websocket/spot/protocol/pb
```

## Endpoint status

MEXC's January 2026 futures-domain announcement moved Contract REST from `https://contract.mexc.com` to `https://api.mexc.com` and stated that the old REST domain would stop being supported on 2026-01-20 (UTC+8). Therefore this module uses `https://api.mexc.com`; `contract.DeprecatedBaseURL` exists only as documentation and is never selected automatically. The current Contract API documentation still identifies `wss://contract.mexc.com/edge` as its native WebSocket endpoint.

## Testing and use restrictions

Tests use fixtures and fake transports and do not contact live MEXC endpoints. This module supports public market data only and does not implement Trading APIs.

Use of this code does not grant permission to commercially use, store, retransmit, redistribute, publish, or create derivative products from MEXC data. Before any production or K4K3RU use, the user must review the [MEXC User Agreement](https://www.mexc.com/ja-JP/terms/), current API documentation, applicable regional restrictions, and obtain any required written consent or data license from MEXC.
