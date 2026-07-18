package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pagedServer serves n pages of one item each, linking each page to the next.
func pagedServer(t *testing.T, pages int) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := 0
		if p := r.URL.Query().Get("page"); p != "" {
			_, _ = fmt.Sscanf(p, "%d", &page)
		}
		resp := map[string]any{
			"value": []map[string]any{{"id": fmt.Sprintf("item-%d", page)}},
		}
		if page < pages-1 {
			resp["@odata.nextLink"] = fmt.Sprintf("%s/me/messages?page=%d", srv.URL, page+1)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newListClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return New(srv.URL, nil, WithMaxRetries(0))
}

func TestList_SinglePageByDefault(t *testing.T) {
	c := newListClient(t, pagedServer(t, 3))
	items, err := c.List(t.Context(), "me/messages", nil, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, items, 1, "without --all/--limit only the first page is fetched")
}

func TestList_AllFollowsNextLink(t *testing.T) {
	c := newListClient(t, pagedServer(t, 3))
	items, err := c.List(t.Context(), "me/messages", nil, ListOptions{All: true})
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestList_LimitTruncates(t *testing.T) {
	c := newListClient(t, pagedServer(t, 5))
	items, err := c.List(t.Context(), "me/messages", nil, ListOptions{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestList_NextLinkHostGuard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"value":[{"id":"1"}],"@odata.nextLink":"https://evil.example.com/steal"}`))
	}))
	defer srv.Close()
	c := New(srv.URL, nil, WithMaxRetries(0))
	_, err := c.List(t.Context(), "me/messages", nil, ListOptions{All: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to follow")
}

func TestList_PageCap(t *testing.T) {
	// A server that always links to another page must hit the cap, not loop forever.
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"value":[{"id":"x"}],"@odata.nextLink":"%s/me/messages"}`, srv.URL)
	}))
	defer srv.Close()
	c := New(srv.URL, nil, WithMaxRetries(0))
	items, err := c.List(t.Context(), "me/messages", nil, ListOptions{All: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pages")
	assert.Len(t, items, pageCap)
}

func TestList_QueryAndHeadersForwarded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "5", r.URL.Query().Get("$top"))
		assert.Equal(t, "eventual", r.Header.Get("ConsistencyLevel"))
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()
	c := New(srv.URL, nil, WithMaxRetries(0))
	q := url.Values{}
	q.Set("$top", "5")
	items, err := c.List(t.Context(), "users", q, ListOptions{Headers: map[string]string{"ConsistencyLevel": "eventual"}})
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestList_DryRunReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("dry-run must not hit the server")
	}))
	defer srv.Close()
	var sink safeBuffer
	c := New(srv.URL, nil, WithDryRun(true, &sink))
	items, err := c.List(t.Context(), "me/messages", nil, ListOptions{All: true})
	require.NoError(t, err)
	assert.Nil(t, items)
}

// safeBuffer is a minimal io.Writer for dry-run capture.
type safeBuffer struct{ b []byte }

func (s *safeBuffer) Write(p []byte) (int, error) { s.b = append(s.b, p...); return len(p), nil }
