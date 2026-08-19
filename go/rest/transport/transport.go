package transport

import (
	"context"
	"net/http"
	"net/url"
)

// Request contains immutable values for one REST request.
type Request struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
}

// Executor executes REST requests.
type Executor interface {
	Do(ctx context.Context, request Request) ([]byte, error)
}
