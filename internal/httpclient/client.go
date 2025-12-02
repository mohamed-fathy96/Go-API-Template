package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"io"
	"kabsa/internal/logging"
	"net/http"
	"net/url"
	"time"
)

// HTTPError represents a non-2xx response with the body captured for debugging.
type HTTPError struct {
	StatusCode int
	Body       []byte
	Message    string
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("http %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("http %d", e.StatusCode)
}

type Client struct {
	baseURL *url.URL
	client  *http.Client
	logger  logging.Logger
}

// New creates an instrumented HTTP client for talking to an external service.
// baseURL should be like "https://orders-api.internal" (no trailing slash).
func New(baseURL string, timeout time.Duration, logger logging.Logger) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse baseURL: %w", err)
	}

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	return &Client{
		baseURL: u,
		client:  httpClient,
		logger:  logger,
	}, nil
}

// buildURL joins the base URL with a relative path and optional query parameters.
func (c *Client) buildURL(path string, query url.Values) (string, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parse path: %w", err)
	}

	u := c.baseURL.ResolveReference(rel)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

// GetJSON performs a GET and decodes the JSON response into out.
// out should be a pointer to a struct/slice/etc.
// If the status code >= 400, it returns *HTTPError.
func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, out any) error {
	urlStr, err := c.buildURL(path, query)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		c.logger.Error("external http error",
			"status", resp.StatusCode,
			"path", path,
		)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       body,
			Message:    string(body),
		}
	}

	if len(body) == 0 || out == nil {
		return nil
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("unmarshal body: %w", err)
	}

	return nil
}

// PostJSON sends a JSON body and decodes a JSON response into out.
// out should be a pointer to a struct/slice/etc.
// If the status code >= 400, it returns *HTTPError.
func (c *Client) PostJSON(ctx context.Context, path string, payload any, out any) error {
	urlStr, err := c.buildURL(path, nil)
	if err != nil {
		return err
	}

	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, body)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		c.logger.Error("external http error",
			"status", resp.StatusCode,
			"path", path,
		)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Body:       respBody,
			Message:    string(respBody),
		}
	}

	if len(respBody) == 0 || out == nil {
		return nil
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("unmarshal body: %w", err)
	}

	return nil
}
