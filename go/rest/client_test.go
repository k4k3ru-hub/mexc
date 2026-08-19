package rest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/mexc/go/rest/transport"
)

type fakeHTTPClient struct {
	response *http.Response
	err      error
}

func (f fakeHTTPClient) Do(*http.Request) (*http.Response, error) { return f.response, f.err }

func TestVenueAndHTTPError(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest} {
		c, err := NewClient(&ClientOption{BaseURL: "http://example.test", HTTPClient: fakeHTTPClient{response: &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"code":-1121,"msg":"bad symbol"}`))}}}, "unused")
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Do(context.Background(), transport.Request{Method: http.MethodGet, Path: "/x"})
		var target *ResponseError
		if !errors.As(err, &target) || target.Code.String() != "-1121" {
			t.Fatalf("status %d: %v", status, err)
		}
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, _ := NewClient(&ClientOption{BaseURL: "http://example.test", HTTPClient: fakeHTTPClient{err: ctx.Err()}}, "unused")
	_, err := c.Do(ctx, transport.Request{Method: http.MethodGet, Path: "/x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error chain lost: %v", err)
	}
}
