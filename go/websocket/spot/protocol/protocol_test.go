package protocol

import (
	"testing"

	"github.com/k4k3ru-hub/mexc/go/websocket/spot/protocol/pb"
	"google.golang.org/protobuf/proto"
)

func TestDecodeOfficialAggregateDealsFixture(t *testing.T) {
	symbol := "BTCUSDT"
	send := int64(123)
	fixture, err := proto.Marshal(&pb.PushDataV3ApiWrapper{Channel: "spot@public.aggre.deals.v3.api.pb@100ms@BTCUSDT", Symbol: &symbol, SendTime: &send, Body: &pb.PushDataV3ApiWrapper_PublicAggreDeals{PublicAggreDeals: &pb.PublicAggreDealsV3Api{EventType: "deals", Deals: []*pb.PublicAggreDealsV3ApiItem{{Price: "1.000000000000000001", Quantity: "2", TradeType: 1, Time: 100, TradeId: "a"}, {Price: "3", Quantity: "4", TradeType: 2, Time: 101, TradeId: "b"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecodeBinary(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Trades.Trades) != 2 || out.Trades.Trades[0].TradeID != "a" || out.Trades.Trades[1].TradeID != "b" {
		t.Fatalf("order lost: %#v", out)
	}
}
func TestDecodeDepthPreservesZeroAndVersions(t *testing.T) {
	symbol := "BTCUSDT"
	fixture, _ := proto.Marshal(&pb.PushDataV3ApiWrapper{Symbol: &symbol, Body: &pb.PushDataV3ApiWrapper_PublicAggreDepths{PublicAggreDepths: &pb.PublicAggreDepthsV3Api{FromVersion: "11", ToVersion: "12", Bids: []*pb.PublicAggreDepthV3ApiItem{{Price: "1", Quantity: "0"}}}}})
	out, err := DecodeBinary(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if out.DiffDepth.Bids[0].Quantity != "0" {
		t.Fatal("zero removed")
	}
	ok, err := out.DiffDepth.IsContinuous("10")
	if err != nil || !ok {
		t.Fatalf("continuity: %v %v", ok, err)
	}
}
func TestDecodePartialDepth(t *testing.T) {
	symbol := "BTCUSDT"
	fixture, _ := proto.Marshal(&pb.PushDataV3ApiWrapper{Symbol: &symbol, Body: &pb.PushDataV3ApiWrapper_PublicLimitDepths{PublicLimitDepths: &pb.PublicLimitDepthsV3Api{Version: "99", Asks: []*pb.PublicLimitDepthV3ApiItem{{Price: "2", Quantity: "3"}}}}})
	out, err := DecodeBinary(fixture)
	if err != nil || out.PartialDepth.Version != "99" {
		t.Fatalf("unexpected: %#v %v", out, err)
	}
}
