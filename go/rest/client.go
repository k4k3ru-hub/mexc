package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/k4k3ru-hub/mexc/go/rest/transport"
)

// HTTPClient executes HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientOption configures a REST transport.
type ClientOption struct {
	BaseURL        string
	ConnectTimeout time.Duration
	HTTPClient     HTTPClient
}

// Client executes MEXC REST requests.
type Client struct {
	baseURL    string
	httpClient HTTPClient
}

// ResponseError represents an HTTP or venue response error.
type ResponseError struct {
	StatusCode int
	Code       json.Number
	Message    string
}

// Error returns the response error message.
//
// Version:
//   - 2026-08-19: Added.
func (e *ResponseError) Error() string {
	if e == nil {
		return "failed to execute rest request: response_error=null"
	}
	if e.Message != "" {
		return fmt.Sprintf("failed to execute rest request: venue rejected request: status_code=%d code=%s message=%q", e.StatusCode, e.Code.String(), e.Message)
	}
	return fmt.Sprintf("failed to execute rest request: unexpected HTTP status: status_code=%d", e.StatusCode)
}

// DefaultClientOption returns REST defaults for a base URL.
//
// Version:
//   - 2026-08-19: Added.
func DefaultClientOption(baseURL string) *ClientOption {
	return &ClientOption{BaseURL: baseURL, ConnectTimeout: 3 * time.Second}
}

// NewClient creates a REST transport without making a network request.
//
// Version:
//   - 2026-08-19: Added.
func NewClient(option *ClientOption, defaultBaseURL string) (*Client, error) {
	if option == nil {
		option = DefaultClientOption(defaultBaseURL)
	}
	baseURL := strings.TrimRight(option.BaseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("failed to create rest client: base_url=empty")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("failed to create rest client: base_url=invalid: %w", err)
	}
	httpClient := option.HTTPClient
	if httpClient == nil {
		timeout := option.ConnectTimeout
		if timeout == 0 {
			timeout = 3 * time.Second
		}
		if timeout < 0 {
			return nil, fmt.Errorf("failed to create rest client: connect_timeout=out_of_range")
		}
		httpClient = &http.Client{Transport: &http.Transport{DialContext: (&net.Dialer{Timeout: timeout}).DialContext}}
	}
	return &Client{baseURL: baseURL, httpClient: httpClient}, nil
}

// Do executes one REST request.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Do(ctx context.Context, request transport.Request) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to execute rest request: client=null")
	}
	if c.httpClient == nil {
		return nil, fmt.Errorf("failed to execute rest request: http_client=null")
	}
	if request.Method == "" {
		return nil, fmt.Errorf("failed to execute rest request: method=empty")
	}
	if request.Path == "" {
		return nil, fmt.Errorf("failed to execute rest request: path=empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	u := c.baseURL + "/" + strings.TrimLeft(request.Path, "/")
	if len(request.Query) > 0 {
		u += "?" + request.Query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, request.Method, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute rest request: failed to create HTTP request: %w", err)
	}
	req.Header = request.Header.Clone()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute rest request: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("failed to execute rest request: response=null")
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("failed to execute rest request: response_body=null")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to execute rest request: failed to read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope struct {
			Code    json.Number `json:"code"`
			Message string      `json:"msg"`
		}
		_ = json.Unmarshal(body, &envelope)
		return nil, &ResponseError{StatusCode: resp.StatusCode, Code: envelope.Code, Message: envelope.Message}
	}
	var venueEnvelope struct {
		Code    json.Number `json:"code"`
		Msg     string      `json:"msg"`
		Message string      `json:"message"`
	}
	if err := json.Unmarshal(body, &venueEnvelope); err == nil && venueEnvelope.Code.String() != "" && venueEnvelope.Code.String() != "0" {
		message := venueEnvelope.Msg
		if message == "" {
			message = venueEnvelope.Message
		}
		return nil, &ResponseError{StatusCode: resp.StatusCode, Code: venueEnvelope.Code, Message: message}
	}
	return body, nil
}
