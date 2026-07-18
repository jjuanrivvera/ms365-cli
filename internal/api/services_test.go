package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMailList_DefaultsAndFolder(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/me/mailFolders/inbox/messages", r.URL.Path)
		assert.Equal(t, mailDefaultSelect, r.URL.Query().Get("$select"))
		assert.Equal(t, "10", r.URL.Query().Get("$top"))
		_, _ = w.Write([]byte(`{"value":[{"id":"m1","subject":"hi"}]}`))
	})
	items, err := c.MailList(t.Context(), MailListOptions{Folder: "inbox", Top: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestMailList_SearchIsQuoted(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, `"from:ana \"urgent\""`, r.URL.Query().Get("$search"))
		assert.Empty(t, r.URL.Query().Get("$filter"))
		_, _ = w.Write([]byte(`{"value":[]}`))
	})
	_, err := c.MailList(t.Context(), MailListOptions{Search: `from:ana "urgent"`})
	require.NoError(t, err)
}

func TestMailList_SearchFilterConflict(t *testing.T) {
	c := New("https://example.invalid", nil)
	_, err := c.MailList(t.Context(), MailListOptions{Search: "a", Filter: "b"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined")
}

func TestMailList_InvalidFolder(t *testing.T) {
	c := New("https://example.invalid", nil)
	_, err := c.MailList(t.Context(), MailListOptions{Folder: "a/b"})
	require.Error(t, err)
}

func TestMailGet_TextPreferHeader(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/me/messages/m-42", r.URL.Path)
		assert.Equal(t, `outlook.body-content-type="text"`, r.Header.Get("Prefer"))
		_, _ = w.Write([]byte(`{"id":"m-42","subject":"hey","body":{"contentType":"text","content":"hello"}}`))
	})
	msg, raw, err := c.MailGet(t.Context(), "m-42", true)
	require.NoError(t, err)
	assert.Equal(t, "hello", msg.Body.Content)
	assert.NotEmpty(t, raw)
}

func TestMailGet_RawKeepsHTML(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Prefer"))
		_, _ = w.Write([]byte(`{"id":"m","body":{"contentType":"html","content":"<b>x</b>"}}`))
	})
	msg, _, err := c.MailGet(t.Context(), "m", false)
	require.NoError(t, err)
	assert.Equal(t, "html", msg.Body.ContentType)
}

func TestMailGet_InvalidID(t *testing.T) {
	c := New("https://example.invalid", nil)
	for _, bad := range []string{"", "  ", "a/b", "a?b", "a#b", `a\b`} {
		_, _, err := c.MailGet(t.Context(), bad, true)
		require.Error(t, err, "id %q must be rejected", bad)
	}
}

func TestCalendarView_WindowAndOrder(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/me/calendarView", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "2026-07-20T00:00:00Z", q.Get("startDateTime"))
		assert.Equal(t, "2026-07-27T00:00:00Z", q.Get("endDateTime"))
		assert.Equal(t, "start/dateTime", q.Get("$orderby"))
		assert.Equal(t, calendarDefaultSelect, q.Get("$select"))
		_, _ = w.Write([]byte(`{"value":[{"id":"e1"}]}`))
	})
	items, err := c.CalendarView(t.Context(), CalendarViewOptions{
		From: mustTime(t, "2026-07-20T00:00:00Z"),
		To:   mustTime(t, "2026-07-27T00:00:00Z"),
	})
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

func TestCalendarView_RejectsInvertedWindow(t *testing.T) {
	c := New("https://example.invalid", nil)
	_, err := c.CalendarView(t.Context(), CalendarViewOptions{
		From: mustTime(t, "2026-07-27T00:00:00Z"),
		To:   mustTime(t, "2026-07-20T00:00:00Z"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be before")
}

func TestEventsList_FilterAndTop(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/me/events", r.URL.Path)
		assert.Equal(t, "isOrganizer eq true", r.URL.Query().Get("$filter"))
		assert.Equal(t, "25", r.URL.Query().Get("$top"))
		_, _ = w.Write([]byte(`{"value":[]}`))
	})
	_, err := c.EventsList(t.Context(), EventListOptions{Top: 25, Filter: "isOrganizer eq true"})
	require.NoError(t, err)
}

func TestMe(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/me", r.URL.Path)
		assert.Equal(t, meSelect, r.URL.Query().Get("$select"))
		_, _ = w.Write([]byte(`{"displayName":"Juan","userPrincipalName":"juan@example.com"}`))
	})
	body, err := c.Me(t.Context())
	require.NoError(t, err)
	assert.Contains(t, string(body), "Juan")
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tm
}
