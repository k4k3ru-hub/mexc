package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestRunSpotExchangeInfo(t *testing.T) {
	httpClient := httpClientFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/v3/exchangeInfo" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("symbol") != "BTCUSDT" || query.Get("symbols") != `["BTCUSDT","ETHUSDT"]` || query.Get("status") != "1" || query.Get("tradeSideType") != "2" {
			t.Fatalf("query = %v", query)
		}
		body := `{"timezone":"UTC","serverTime":1750000000000,"symbols":[{"symbol":"BTCUSDT","status":"1","baseAsset":"BTC","quoteAsset":"USDT","maxQuoteAmount":"9007199254740993.123456789"}]}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(context.Background(), []string{"rest", "spot", "exchange-info", "--symbol", "BTCUSDT", "--symbols", "BTCUSDT,ETHUSDT", "--status", "1", "--trade-side-type", "2", "--base-url", "http://mexc.test"}, &stdout, &stderr, &Option{HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"symbol": "BTCUSDT"`) || !strings.Contains(stdout.String(), `"maxQuoteAmount": "9007199254740993.123456789"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestRunValidation(t *testing.T) {
	var output bytes.Buffer
	if err := Run(context.Background(), []string{"unknown"}, &output, &output, nil); err == nil {
		t.Fatal("expected command error")
	}
	if err := Run(context.Background(), []string{"rest", "spot", "exchange-info", "--symbols", "BTCUSDT,,ETHUSDT"}, &output, &output, nil); err == nil {
		t.Fatal("expected symbols error")
	}
	if err := Run(context.Background(), nil, nil, &output, nil); err == nil {
		t.Fatal("expected stdout error")
	}
}
