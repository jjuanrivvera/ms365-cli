package api

import (
	"encoding/json"
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

func TestMailSend_PayloadShape(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/me/sendMail", r.URL.Path)
		var got map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, true, got["saveToSentItems"])
		msg := got["message"].(map[string]any)
		assert.Equal(t, "Lunch?", msg["subject"])
		body := msg["body"].(map[string]any)
		assert.Equal(t, "text", body["contentType"])
		assert.Equal(t, "12:30?", body["content"])
		to := msg["toRecipients"].([]any)
		require.Len(t, to, 2)
		first := to[0].(map[string]any)["emailAddress"].(map[string]any)
		assert.Equal(t, "ana@example.com", first["address"])
		cc := msg["ccRecipients"].([]any)
		require.Len(t, cc, 1)
		_, hasBcc := msg["bccRecipients"]
		assert.False(t, hasBcc, "empty bcc must be omitted, not sent as []")
		w.WriteHeader(http.StatusAccepted)
	})
	err := c.MailSend(t.Context(), MailSendOptions{
		To:         []string{"ana@example.com", " bo@example.com "},
		Cc:         []string{"boss@example.com"},
		Subject:    "Lunch?",
		Body:       "12:30?",
		SaveToSent: true,
	})
	require.NoError(t, err)
}

func TestMailSend_HTMLAndNoSave(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, false, got["saveToSentItems"])
		body := got["message"].(map[string]any)["body"].(map[string]any)
		assert.Equal(t, "html", body["contentType"])
		w.WriteHeader(http.StatusAccepted)
	})
	err := c.MailSend(t.Context(), MailSendOptions{To: []string{"a@x.com"}, Body: "<b>x</b>", HTML: true})
	require.NoError(t, err)
}

func TestMailSend_Validation(t *testing.T) {
	c := New("https://example.invalid", nil)
	err := c.MailSend(t.Context(), MailSendOptions{Subject: "no recipients"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--to")

	err = c.MailSend(t.Context(), MailSendOptions{To: []string{"not-an-address"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recipient")
}

func TestMailReply_ReplyAndReplyAll(t *testing.T) {
	var gotPath, gotComment string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var got map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		gotComment = got["comment"]
		w.WriteHeader(http.StatusAccepted)
	})

	require.NoError(t, c.MailReply(t.Context(), "m-1", "ack", false))
	assert.Equal(t, "/me/messages/m-1/reply", gotPath)
	assert.Equal(t, "ack", gotComment)

	require.NoError(t, c.MailReply(t.Context(), "m-1", "ack all", true))
	assert.Equal(t, "/me/messages/m-1/replyAll", gotPath)
	assert.Equal(t, "ack all", gotComment)
}

func TestMailReply_InvalidID(t *testing.T) {
	c := New("https://example.invalid", nil)
	require.Error(t, c.MailReply(t.Context(), "a/b", "x", false))
}

func TestEventCreate_PayloadShape(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/me/events", r.URL.Path)
		var got map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, "1:1 Ana", got["subject"])
		start := got["start"].(map[string]any)
		assert.Equal(t, "2026-07-21T10:00:00", start["dateTime"])
		assert.Equal(t, "America/Caracas", start["timeZone"])
		loc := got["location"].(map[string]any)
		assert.Equal(t, "Room 3", loc["displayName"])
		body := got["body"].(map[string]any)
		assert.Equal(t, "text", body["contentType"])
		att := got["attendees"].([]any)
		require.Len(t, att, 1)
		a := att[0].(map[string]any)
		assert.Equal(t, "required", a["type"])
		assert.Equal(t, true, got["isOnlineMeeting"])
		assert.Equal(t, "teamsForBusiness", got["onlineMeetingProvider"])
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"e-new","subject":"1:1 Ana"}`))
	})
	subject, location, body := "1:1 Ana", "Room 3", "agenda"
	online := true
	created, err := c.EventCreate(t.Context(), EventFields{
		Subject:       &subject,
		Start:         &EventDateTime{DateTime: "2026-07-21T10:00:00", TimeZone: "America/Caracas"},
		End:           &EventDateTime{DateTime: "2026-07-21T10:30:00", TimeZone: "America/Caracas"},
		Location:      &location,
		Body:          &body,
		Attendees:     []string{"ana@example.com"},
		OnlineMeeting: &online,
	})
	require.NoError(t, err)
	assert.Contains(t, string(created), "e-new")
}

func TestEventCreate_RequiredFields(t *testing.T) {
	c := New("https://example.invalid", nil)
	subject := "x"
	_, err := c.EventCreate(t.Context(), EventFields{Subject: &subject})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestEventUpdate_SendsOnlySetFields(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/me/events/e-1", r.URL.Path)
		var got map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, map[string]any{"subject": "moved"}, got, "unset fields must not be PATCHed")
		_, _ = w.Write([]byte(`{"id":"e-1","subject":"moved"}`))
	})
	subject := "moved"
	updated, err := c.EventUpdate(t.Context(), "e-1", EventFields{Subject: &subject})
	require.NoError(t, err)
	assert.Contains(t, string(updated), "moved")
}

func TestEventUpdate_OnlineMeetingFalseOmitsProvider(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, false, got["isOnlineMeeting"])
		_, hasProvider := got["onlineMeetingProvider"]
		assert.False(t, hasProvider)
		_, _ = w.Write([]byte(`{"id":"e-1"}`))
	})
	online := false
	_, err := c.EventUpdate(t.Context(), "e-1", EventFields{OnlineMeeting: &online})
	require.NoError(t, err)
}

func TestEventUpdate_Validation(t *testing.T) {
	c := New("https://example.invalid", nil)
	_, err := c.EventUpdate(t.Context(), "a/b", EventFields{})
	require.Error(t, err)

	_, err = c.EventUpdate(t.Context(), "e-1", EventFields{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
}

func TestEventDelete(t *testing.T) {
	var gotMethod, gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	require.NoError(t, c.EventDelete(t.Context(), "e-1"))
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/me/events/e-1", gotPath)

	require.Error(t, c.EventDelete(t.Context(), "a/b"))
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tm
}
