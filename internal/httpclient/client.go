package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	requestTimeout = 20 * time.Second
	maxBodyBytes   = 32 << 20
	userAgent      = "Mozilla/5.0 (compatible; newsbot/1.0)"
	acceptTypes    = "application/json"
	shortestHold   = time.Minute
)

type StatusError struct {
	Code  int
	URL   string
	Retry time.Duration
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("GET %s: HTTP %d", e.URL, e.Code)
}

func (e *StatusError) RetryAfter() time.Duration {
	return e.Retry
}

type HeldError struct {
	Host  string
	Until time.Time
}

func (e *HeldError) Error() string {
	return fmt.Sprintf("%s asked to wait until %s", e.Host, e.Until.UTC().Format(time.TimeOnly))
}

type Client struct {
	http *http.Client
	mu   sync.Mutex
	held map[string]time.Time
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: requestTimeout}, held: map[string]time.Time{}}
}

func (c *Client) waiting(host string) (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	until := c.held[host]

	return until, time.Now().Before(until)
}

func (c *Client) hold(host string, asked time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.held[host] = time.Now().Add(max(asked, shortestHold))
}

func (c *Client) Bytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if until, held := c.waiting(req.URL.Host); held {
		return nil, &HeldError{Host: req.URL.Host, Until: until}
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", acceptTypes)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		refused := statusError(resp, url)

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusTeapot {
			c.hold(req.URL.Host, refused.Retry)
		}

		return nil, refused
	}

	return body, nil
}

func (c *Client) JSON(ctx context.Context, url string, v any) error {
	body, err := c.Bytes(ctx, url)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", url, err)
	}

	return nil
}

func statusError(resp *http.Response, url string) *StatusError {
	se := &StatusError{Code: resp.StatusCode, URL: url}

	if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
		se.Retry = time.Duration(seconds) * time.Second
	}

	return se
}
