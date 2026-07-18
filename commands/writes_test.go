package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jjuanrivvera/ms365-cli/internal/auth"
)

// signInScoped seeds a session whose token carries DefaultScopes PLUS extra — as if the
// user ran `auth login --scopes …` for this profile.
func (e *env) signInScoped(profile string, extra ...string) {
	e.auth.set(profile, auth.Token{
		AccessToken: "fake-token-" + profile,
		ExpiresOn:   time.Now().Add(time.Hour),
		Scopes:      append(append([]string{}, auth.DefaultScopes...), extra...),
		Username:    profile + "@example.com",
		TenantID:    "tenant-" + profile,
	})
}

func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	b, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func TestMailSend_HappyPath(t *testing.T) {
	var got map[string]any
	var path string
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		got = decodeBody(t, r)
		w.WriteHeader(http.StatusAccepted)
	})
	e.signInScoped("default", "Mail.Send")

	out, _, err := e.run("mail", "send",
		"--to", "ana@example.com,bo@example.com",
		"--cc", "boss@example.com",
		"--subject", "Lunch?",
		"--body", "12:30 at the usual place")
	require.NoError(t, err)
	assert.Equal(t, "POST /me/sendMail", path)
	assert.Contains(t, out, `Sent "Lunch?" to ana@example.com, bo@example.com.`)

	assert.Equal(t, true, got["saveToSentItems"], "--save-to-sent defaults to true")
	msg := got["message"].(map[string]any)
	assert.Equal(t, "Lunch?", msg["subject"])
	assert.Len(t, msg["toRecipients"].([]any), 2)
	assert.Len(t, msg["ccRecipients"].([]any), 1)
	body := msg["body"].(map[string]any)
	assert.Equal(t, "text", body["contentType"])
}

func TestMailSend_HTMLAndNoSave(t *testing.T) {
	var got map[string]any
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		got = decodeBody(t, r)
		w.WriteHeader(http.StatusAccepted)
	})
	e.signInScoped("default", "Mail.Send")

	_, _, err := e.run("mail", "send", "--to", "a@x.com", "--subject", "s",
		"--html", "--body", "<b>x</b>", "--save-to-sent=false")
	require.NoError(t, err)
	assert.Equal(t, false, got["saveToSentItems"])
	body := got["message"].(map[string]any)["body"].(map[string]any)
	assert.Equal(t, "html", body["contentType"])
}

func TestMailSend_BodyFile(t *testing.T) {
	var got map[string]any
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		got = decodeBody(t, r)
		w.WriteHeader(http.StatusAccepted)
	})
	e.signInScoped("default", "Mail.Send")

	f := filepath.Join(t.TempDir(), "body.txt")
	require.NoError(t, os.WriteFile(f, []byte("body from file"), 0o600))
	_, _, err := e.run("mail", "send", "--to", "a@x.com", "--subject", "s", "--body-file", f)
	require.NoError(t, err)
	body := got["message"].(map[string]any)["body"].(map[string]any)
	assert.Equal(t, "body from file", body["content"])

	// --body and --body-file together is a user mistake, not a merge.
	_, _, err = e.run("mail", "send", "--to", "a@x.com", "--body", "x", "--body-file", f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestMailSend_RequiresRecipient(t *testing.T) {
	e := newEnv(t, jsonHandler(`{}`))
	e.signInScoped("default", "Mail.Send")
	_, _, err := e.run("mail", "send", "--subject", "no recipients")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--to")
}

func TestMailSend_MissingScopeHint(t *testing.T) {
	e := newEnv(t, func(http.ResponseWriter, *http.Request) {
		t.Error("a scope-blocked send must never reach Graph")
	})
	e.signIn("default") // v1 default scopes only — no Mail.Send
	_, _, err := e.run("mail", "send", "--to", "a@x.com", "--subject", "s", "--body", "b")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run: ms365 auth login -a default --scopes Mail.Send")
}

func TestMailSend_DryRunNeedsNoScope(t *testing.T) {
	e := newEnv(t, func(http.ResponseWriter, *http.Request) {
		t.Error("dry-run must not hit the server")
	})
	e.signIn("default") // no Mail.Send — dry-run must still work
	d := e.deps()
	var buf strings.Builder
	d.out = &buf
	_, _, err := runWithDeps(t, d, "mail", "send", "--to", "a@x.com", "--subject", "s", "--body", "b", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "curl -X POST")
	assert.Contains(t, buf.String(), "me/sendMail")
	assert.Contains(t, buf.String(), "Bearer REDACTED")
}

func TestMailReply_ReplyAndReplyAll(t *testing.T) {
	var path string
	var got map[string]any
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		got = decodeBody(t, r)
		w.WriteHeader(http.StatusAccepted)
	})
	e.signInScoped("default", "Mail.Send")

	out, _, err := e.run("mail", "reply", "m-1", "--body", "works for me")
	require.NoError(t, err)
	assert.Equal(t, "/me/messages/m-1/reply", path)
	assert.Equal(t, "works for me", got["comment"])
	assert.Contains(t, out, "Replied to sender of message m-1.")

	out, _, err = e.run("mail", "reply", "m-1", "--all", "--body", "ack everyone")
	require.NoError(t, err)
	assert.Equal(t, "/me/messages/m-1/replyAll", path)
	assert.Contains(t, out, "Replied to all recipients of message m-1.")
}

func TestMailReply_RequiresBody(t *testing.T) {
	e := newEnv(t, jsonHandler(`{}`))
	e.signInScoped("default", "Mail.Send")
	_, _, err := e.run("mail", "reply", "m-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--body")
}

func TestCalendarCreate_HappyPath(t *testing.T) {
	var path string
	var got map[string]any
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		got = decodeBody(t, r)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"e-new","subject":"1:1 Ana","start":{"dateTime":"2026-07-21T10:00:00","timeZone":"America/Caracas"}}`))
	})
	e.signInScoped("default", "Calendars.ReadWrite")

	out, _, err := e.run("calendar", "create",
		"--subject", "1:1 Ana",
		"--from", "2026-07-21T10:00", "--to", "2026-07-21T10:30",
		"--timezone", "America/Caracas",
		"--location", "Room 3",
		"--attendee", "ana@x.com", "--attendee", "bo@x.com",
		"--online-meeting")
	require.NoError(t, err)
	assert.Equal(t, "POST /me/events", path)
	assert.Contains(t, out, "1:1 Ana")

	assert.Equal(t, "1:1 Ana", got["subject"])
	start := got["start"].(map[string]any)
	assert.Equal(t, "2026-07-21T10:00:00", start["dateTime"])
	assert.Equal(t, "America/Caracas", start["timeZone"])
	end := got["end"].(map[string]any)
	assert.Equal(t, "2026-07-21T10:30:00", end["dateTime"])
	assert.Equal(t, "Room 3", got["location"].(map[string]any)["displayName"])
	assert.Len(t, got["attendees"].([]any), 2)
	assert.Equal(t, true, got["isOnlineMeeting"])
	assert.Equal(t, "teamsForBusiness", got["onlineMeetingProvider"])
}

func TestCalendarCreate_DefaultsToUTC(t *testing.T) {
	var got map[string]any
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		got = decodeBody(t, r)
		_, _ = w.Write([]byte(`{"id":"e-new"}`))
	})
	e.signInScoped("default", "Calendars.ReadWrite")
	_, _, err := e.run("calendar", "create", "--subject", "s",
		"--from", "2026-07-21", "--to", "2026-07-22")
	require.NoError(t, err)
	assert.Equal(t, "UTC", got["start"].(map[string]any)["timeZone"])
	assert.Equal(t, "2026-07-21T00:00:00", got["start"].(map[string]any)["dateTime"])
}

func TestCalendarCreate_Validation(t *testing.T) {
	e := newEnv(t, jsonHandler(`{}`))
	e.signInScoped("default", "Calendars.ReadWrite")

	_, _, err := e.run("calendar", "create", "--subject", "s")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	_, _, err = e.run("calendar", "create", "--subject", "s", "--from", "next tuesday", "--to", "2026-07-22")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from")
}

func TestCalendarUpdate_SendsOnlySetFlags(t *testing.T) {
	var path string
	var got map[string]any
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		got = decodeBody(t, r)
		_, _ = w.Write([]byte(`{"id":"e-1","subject":"moved"}`))
	})
	e.signInScoped("default", "Calendars.ReadWrite")

	out, _, err := e.run("calendar", "update", "e-1", "--subject", "moved")
	require.NoError(t, err)
	assert.Equal(t, "PATCH /me/events/e-1", path)
	assert.Equal(t, map[string]any{"subject": "moved"}, got, "unset flags must not be PATCHed")
	assert.Contains(t, out, "moved")
}

func TestCalendarUpdate_NothingToUpdate(t *testing.T) {
	e := newEnv(t, jsonHandler(`{}`))
	e.signInScoped("default", "Calendars.ReadWrite")
	_, _, err := e.run("calendar", "update", "e-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing to update")
}

func TestCalendarDelete_WithYes(t *testing.T) {
	var path string
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	e.signInScoped("default", "Calendars.ReadWrite")

	out, _, err := e.run("calendar", "delete", "e-1", "--yes")
	require.NoError(t, err)
	assert.Equal(t, "DELETE /me/events/e-1", path)
	assert.Contains(t, out, "Deleted event e-1.")
}

func TestCalendarDelete_InteractiveConfirm(t *testing.T) {
	hit := false
	e := newEnv(t, func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	})
	e.signInScoped("default", "Calendars.ReadWrite")

	// Confirmed with "y".
	d := e.deps()
	root := newRootCmd(d)
	root.SetArgs([]string{"calendar", "delete", "e-1"})
	root.SetIn(strings.NewReader("y\n"))
	var out, errB strings.Builder
	root.SetOut(&out)
	root.SetErr(&errB)
	require.NoError(t, root.ExecuteContext(t.Context()))
	assert.True(t, hit, "confirmed delete must reach Graph")
	assert.Contains(t, errB.String(), "Delete event e-1?")

	// Declined: anything but y/yes aborts with a non-zero exit and no request.
	hit = false
	root = newRootCmd(e.deps())
	root.SetArgs([]string{"calendar", "delete", "e-1"})
	root.SetIn(strings.NewReader("n\n"))
	root.SetOut(&out)
	root.SetErr(&errB)
	err := root.ExecuteContext(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")
	assert.False(t, hit, "declined delete must not reach Graph")
}

func TestCalendarWrite_MissingScopeHint(t *testing.T) {
	e := newEnv(t, func(http.ResponseWriter, *http.Request) {
		t.Error("a scope-blocked write must never reach Graph")
	})
	e.signIn("default") // v1 default scopes only — no Calendars.ReadWrite
	_, _, err := e.run("calendar", "delete", "e-1", "--yes")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run: ms365 auth login -a default --scopes Calendars.ReadWrite")
}

func TestCalendarDelete_DryRunSkipsConfirm(t *testing.T) {
	e := newEnv(t, func(http.ResponseWriter, *http.Request) {
		t.Error("dry-run must not hit the server")
	})
	e.signIn("default")
	d := e.deps()
	var buf strings.Builder
	d.out = &buf
	// No --yes and no stdin: dry-run must not prompt (it sends nothing).
	_, _, err := runWithDeps(t, d, "calendar", "delete", "e-1", "--dry-run")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "curl -X DELETE")
	assert.Contains(t, buf.String(), "me/events/e-1")
}
