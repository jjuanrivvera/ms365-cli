// Package api is the Microsoft Graph client core: bearer auth via a token source,
// idempotent-only retry honoring Retry-After, @odata.nextLink pagination, a dry-run curl
// mode, and typed mail/calendar/user services on top of one generic request path.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// DefaultBaseURL is the Microsoft Graph v1.0 endpoint for the worldwide cloud. Sovereign
// clouds (graph.microsoft.us, microsoftgraph.chinacloudapi.cn) override it via --base-url.
const DefaultBaseURL = "https://graph.microsoft.com/v1.0"

// TokenFunc supplies a bearer access token per request. The auth layer backs it with
// MSAL AcquireTokenSilent, so tokens refresh transparently until the refresh token dies.
type TokenFunc func(ctx context.Context) (string, error)

// Client is a Microsoft Graph HTTP client.
type Client struct {
	baseURL string
	token   TokenFunc
	httpc   *http.Client

	// DryRun prints the equivalent curl to DryRunOut instead of sending the request.
	DryRun    bool
	DryRunOut io.Writer
	// ShowToken reveals the bearer token in dry-run output (redacted by default).
	ShowToken bool

	Verbose    bool
	VerboseOut io.Writer

	// Timezone sets `Prefer: outlook.timezone="..."` so mail/calendar datetimes come back
	// in the user's zone instead of UTC.
	Timezone string

	maxRetries int
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient overrides the HTTP transport (tests point it at httptest servers).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpc = h } }

// WithDryRun enables curl-printing mode.
func WithDryRun(dry bool, out io.Writer) Option {
	return func(c *Client) { c.DryRun = dry; c.DryRunOut = out }
}

// WithTimezone sets the Prefer outlook.timezone header value.
func WithTimezone(tz string) Option { return func(c *Client) { c.Timezone = tz } }

// WithMaxRetries overrides the retry budget (tests set 0 for speed).
func WithMaxRetries(n int) Option { return func(c *Client) { c.maxRetries = n } }

// New builds a Graph client. An empty baseURL means DefaultBaseURL; token may be nil for
// dry-run-only use.
func New(baseURL string, token TokenFunc, opts ...Option) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpc:      http.DefaultClient,
		DryRunOut:  os.Stdout,
		VerboseOut: os.Stderr,
		maxRetries: 3,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// BaseURL returns the resolved Graph base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// GetJSON GETs path (relative to the base URL) and decodes the JSON response into out.
// A nil out discards the body. extra headers are per-request (e.g. body-content-type).
func (c *Client) GetJSON(ctx context.Context, path string, q url.Values, extra map[string]string, out any) error {
	status, _, body, err := c.Do(ctx, http.MethodGet, path, q, nil, extra)
	if err != nil {
		return err
	}
	if status == 0 { // dry-run: no response to decode
		return nil
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode Graph response: %w", err)
	}
	return nil
}

// Do sends one authenticated request and returns status, headers, and body. A dry-run
// returns status 0 with no error. Non-2xx statuses return an *APIError.
func (c *Client) Do(ctx context.Context, method, path string, q url.Values, body []byte, extra map[string]string) (int, http.Header, []byte, error) {
	u := c.baseURL + "/" + strings.TrimLeft(path, "/")
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return c.doURL(ctx, method, u, body, extra)
}

// doURL is the request path shared by Do and the @odata.nextLink follower (which receives
// a full URL from the server).
func (c *Client) doURL(ctx context.Context, method, fullURL string, body []byte, extra map[string]string) (int, http.Header, []byte, error) {
	headers := map[string]string{
		"Accept": "application/json",
	}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	if c.Timezone != "" {
		headers["Prefer"] = `outlook.timezone="` + c.Timezone + `"`
	}
	for k, v := range extra {
		if v == "" {
			delete(headers, k)
			continue
		}
		// Multiple Prefer values are comma-joined per RFC 7240.
		if k == "Prefer" && headers["Prefer"] != "" {
			headers["Prefer"] = headers["Prefer"] + ", " + v
			continue
		}
		headers[k] = v
	}

	if c.DryRun {
		c.printCurl(ctx, method, fullURL, body, headers)
		return 0, nil, nil, nil
	}

	tok := ""
	if c.token != nil {
		t, err := c.token(ctx)
		if err != nil {
			return 0, nil, nil, err
		}
		tok = t
	}

	send := func() (*http.Response, error) {
		var rdr io.Reader
		if body != nil {
			rdr = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, fullURL, rdr)
		if err != nil {
			return nil, err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		if c.Verbose {
			fmt.Fprintf(c.VerboseOut, "> %s %s\n", method, fullURL)
		}
		return c.httpc.Do(req)
	}

	resp, err := c.sendWithRetry(ctx, method, send)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return 0, nil, nil, fmt.Errorf("read response: %w", err)
	}
	if c.Verbose {
		fmt.Fprintf(c.VerboseOut, "< HTTP %d (%d bytes)\n", resp.StatusCode, len(respBody))
	}
	if resp.StatusCode >= 400 {
		return resp.StatusCode, resp.Header, respBody, parseAPIError(resp.StatusCode, respBody, resp.Header)
	}
	return resp.StatusCode, resp.Header, respBody, nil
}

// printCurl emits a copy-pasteable curl equivalent with the token redacted unless
// --show-token was given.
func (c *Client) printCurl(ctx context.Context, method, fullURL string, body []byte, headers map[string]string) {
	var b strings.Builder
	b.WriteString("curl -X " + method + " " + shellQuote(fullURL))
	tok := "REDACTED"
	if c.ShowToken && c.token != nil {
		if t, err := c.token(ctx); err == nil && t != "" {
			tok = t
		}
	}
	b.WriteString(" \\\n  -H " + shellQuote("Authorization: Bearer "+tok))
	for _, k := range sortedKeys(headers) {
		b.WriteString(" \\\n  -H " + shellQuote(k+": "+headers[k]))
	}
	if body != nil {
		b.WriteString(" \\\n  -d " + shellQuote(string(body)))
	}
	fmt.Fprintln(c.DryRunOut, b.String())
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Deterministic dry-run output — never map-iteration order.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
