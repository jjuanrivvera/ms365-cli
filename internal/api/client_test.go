package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient spins an httptest server and a client pointed at it.
func newTestClient(t *testing.T, handler http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	token := func(context.Context) (string, error) { return "test-token", nil }
	return New(srv.URL, token, append([]Option{WithMaxRetries(0)}, opts...)...)
}

func TestClient_GetJSON_SendsAuthAndHeaders(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, `outlook.timezone="America/Caracas"`, r.Header.Get("Prefer"))
		assert.Equal(t, "/me", r.URL.Path)
		_, _ = w.Write([]byte(`{"displayName":"Juan"}`))
	}, WithTimezone("America/Caracas"))

	var out struct {
		DisplayName string `json:"displayName"`
	}
	require.NoError(t, c.GetJSON(t.Context(), "me", nil, nil, &out))
	assert.Equal(t, "Juan", out.DisplayName)
}

func TestClient_DefaultBaseURL(t *testing.T) {
	c := New("", nil)
	assert.Equal(t, DefaultBaseURL, c.BaseURL())
}

func TestClient_TokenError(t *testing.T) {
	c := New("http://127.0.0.1:0", func(context.Context) (string, error) {
		return "", assert.AnError
	})
	err := c.GetJSON(t.Context(), "me", nil, nil, nil)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestClient_APIError_HintsByStatus(t *testing.T) {
	cases := []struct {
		status int
		code   string
		hint   string
	}{
		{401, "InvalidAuthenticationToken", "auth login"},
		{403, "ErrorAccessDenied", "consent"},
		{404, "ErrorItemNotFound", "list"},
		{429, "TooManyRequests", "throttled"},
		{400, "BadRequest", "$filter"},
		{503, "ServiceUnavailable", "transient"},
	}
	for _, tc := range cases {
		c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"error":{"code":"` + tc.code + `","message":"nope","innerError":{"request-id":"rid-1"}}}`))
		})
		err := c.GetJSON(t.Context(), "me", nil, nil, nil)
		require.Error(t, err, "status %d", tc.status)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, tc.status, apiErr.StatusCode)
		assert.Equal(t, tc.code, apiErr.Code)
		assert.Equal(t, "rid-1", apiErr.RequestID)
		assert.Contains(t, err.Error(), tc.hint, "status %d should hint", tc.status)
	}
}

func TestClient_APIError_NonJSONBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("request-id", "hdr-rid")
		w.WriteHeader(502)
		_, _ = w.Write([]byte("Bad Gateway"))
	})
	err := c.GetJSON(t.Context(), "me", nil, nil, nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "hdr-rid", apiErr.RequestID)
	assert.Equal(t, "Bad Gateway", apiErr.Message)
}

func TestClient_DryRun_PrintsCurlRedacted(t *testing.T) {
	var buf bytes.Buffer
	token := func(context.Context) (string, error) { return "SECRET", nil }
	c := New("https://graph.example.com/v1.0", token, WithDryRun(true, &buf))

	q := url.Values{}
	q.Set("$top", "5")
	status, _, _, err := c.Do(t.Context(), "GET", "me/messages", q, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, status)
	out := buf.String()
	assert.Contains(t, out, "curl -X GET")
	assert.Contains(t, out, "graph.example.com")
	assert.Contains(t, out, "Bearer REDACTED")
	assert.NotContains(t, out, "SECRET")
}

func TestClient_DryRun_ShowToken(t *testing.T) {
	var buf bytes.Buffer
	token := func(context.Context) (string, error) { return "SECRET", nil }
	c := New("https://graph.example.com/v1.0", token, WithDryRun(true, &buf))
	c.ShowToken = true
	_, _, _, err := c.Do(t.Context(), "POST", "me/sendMail", nil, []byte(`{"a":1}`), nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Bearer SECRET")
	assert.Contains(t, buf.String(), `-d '{"a":1}'`)
	assert.Contains(t, buf.String(), "Content-Type: application/json")
}

func TestRetry_429HonorsRetryAfterSeconds(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := New(srv.URL, nil, WithMaxRetries(2))
	require.NoError(t, c.GetJSON(t.Context(), "me", nil, nil, nil))
	assert.Equal(t, 2, calls)
}

func TestRetry_5xxRetriesGET(t *testing.T) {
	old := retryBase
	retryBase = time.Millisecond
	defer func() { retryBase = old }()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(500)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := New(srv.URL, nil, WithMaxRetries(3))
	require.NoError(t, c.GetJSON(t.Context(), "me", nil, nil, nil))
	assert.Equal(t, 3, calls)
}

func TestRetry_POSTNeverRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":{"code":"x","message":"boom"}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, nil, WithMaxRetries(3))
	_, _, _, err := c.Do(t.Context(), "POST", "me/sendMail", nil, []byte(`{}`), nil)
	require.Error(t, err)
	assert.Equal(t, 1, calls, "POST must not be auto-retried")
}

func TestRetry_ExhaustionReturnsLastResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":{"code":"TooManyRequests","message":"slow down"}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, nil, WithMaxRetries(1))
	err := c.GetJSON(t.Context(), "me", nil, nil, nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 429, apiErr.StatusCode)
}

func TestRetryAfter_HTTPDate(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", time.Now().Add(2*time.Second).UTC().Format(http.TimeFormat))
	d := retryAfter(h)
	assert.Greater(t, d, time.Duration(0))
	assert.LessOrEqual(t, d, 3*time.Second)

	h.Set("Retry-After", "not-a-date")
	assert.Equal(t, time.Duration(0), retryAfter(h))

	h.Del("Retry-After")
	assert.Equal(t, time.Duration(0), retryAfter(h))
}

func TestIsTransient(t *testing.T) {
	assert.False(t, isTransient(context.Canceled))
	assert.False(t, isTransient(assert.AnError))
}

func TestRetry_ContextCancelDuringBackoff(t *testing.T) {
	old := retryBase
	retryBase = time.Hour // force a long backoff so cancellation is what ends it
	defer func() { retryBase = old }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	c := New(srv.URL, nil, WithMaxRetries(2))

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	err := c.GetJSON(ctx, "me", nil, nil, nil)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, `'a b'`, shellQuote("a b"))
	assert.Equal(t, `'a'\''b'`, shellQuote("a'b"))
}
