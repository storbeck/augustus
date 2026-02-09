package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"
)

// Option configures a Client.
type Option func(*Client)

// Client is a small JSON-first HTTP helper used by generators.
type Client struct {
	httpClient *stdhttp.Client
	baseURL    string
	userAgent  string
	bearer     string
}

// Response wraps an HTTP response with a buffered body for repeatable JSON decoding.
type Response struct {
	StatusCode int
	Headers    stdhttp.Header
	body       []byte
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithBaseURL sets a base URL used when request URLs are relative.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithBearerToken sets Authorization: Bearer <token>.
func WithBearerToken(token string) Option {
	return func(c *Client) {
		c.bearer = token
	}
}

// NewClient constructs a Client with optional configuration.
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &stdhttp.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Post sends a JSON POST request and returns a buffered response.
func (c *Client) Post(ctx context.Context, url string, payload any) (*Response, error) {
	reqURL, err := c.resolveURL(url)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody := new(bytes.Buffer)
	if _, err := respBody.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
		body:       respBody.Bytes(),
	}, nil
}

// JSON decodes the buffered response body into v.
func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.body, v)
}

func (c *Client) resolveURL(url string) (string, error) {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url, nil
	}
	if c.baseURL == "" {
		return "", fmt.Errorf("relative URL %q requires base URL", url)
	}
	if strings.HasPrefix(url, "/") {
		return c.baseURL + url, nil
	}
	return c.baseURL + "/" + url, nil
}
