package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// pageCap bounds @odata.nextLink auto-following so `--all` can never loop forever on a
// misbehaving server.
const pageCap = 50

// listPage is Graph's collection envelope.
type listPage struct {
	Value    []json.RawMessage `json:"value"`
	NextLink string            `json:"@odata.nextLink"`
}

// ListOptions control the generic collection walker.
type ListOptions struct {
	// Limit caps the total items returned (0 = one server page unless All).
	Limit int
	// All follows @odata.nextLink until exhausted (bounded by pageCap).
	All bool
	// Extra headers per request (e.g. ConsistencyLevel: eventual for directory $search).
	Headers map[string]string
}

// List GETs a Graph collection and follows @odata.nextLink per opts. Items come back raw
// so one renderer serves every resource.
func (c *Client) List(ctx context.Context, path string, q url.Values, opts ListOptions) ([]json.RawMessage, error) {
	var items []json.RawMessage
	nextURL := ""
	for page := 0; ; page++ {
		if page >= pageCap {
			return items, fmt.Errorf("stopped after %d pages — narrow the query or use --limit", pageCap)
		}
		var lp listPage
		if nextURL == "" {
			status, _, body, err := c.Do(ctx, "GET", path, q, nil, opts.Headers)
			if err != nil {
				return nil, err
			}
			if status == 0 { // dry-run prints the first request only
				return nil, nil
			}
			if err := json.Unmarshal(body, &lp); err != nil {
				return nil, fmt.Errorf("decode Graph collection: %w", err)
			}
		} else {
			status, _, body, err := c.doURL(ctx, "GET", nextURL, nil, opts.Headers)
			if err != nil {
				return nil, err
			}
			if status == 0 {
				return nil, nil
			}
			if err := json.Unmarshal(body, &lp); err != nil {
				return nil, fmt.Errorf("decode Graph collection: %w", err)
			}
		}

		items = append(items, lp.Value...)
		if opts.Limit > 0 && len(items) >= opts.Limit {
			return items[:opts.Limit], nil
		}
		if lp.NextLink == "" {
			return items, nil
		}
		if !opts.All && opts.Limit == 0 {
			// Single-page mode: surface that more exists without silently fetching it.
			return items, nil
		}
		next, err := c.checkNextLink(lp.NextLink)
		if err != nil {
			return nil, err
		}
		nextURL = next
	}
}

// checkNextLink refuses to follow a nextLink that points off the configured Graph host —
// a defense against a compromised/malicious response redirecting the bearer token.
func (c *Client) checkNextLink(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("invalid @odata.nextLink: %w", err)
	}
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(u.Host, base.Host) {
		return "", fmt.Errorf("@odata.nextLink host %q does not match Graph host %q — refusing to follow", u.Host, base.Host)
	}
	return link, nil
}
