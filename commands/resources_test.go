package commands

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mailListBody = `{"value":[
  {"id":"m1","subject":"Invoice July","from":{"emailAddress":{"name":"Ana","address":"ana@example.com"}},"receivedDateTime":"2026-07-17T10:00:00Z","isRead":false},
  {"id":"m2","subject":"Standup notes","from":{"emailAddress":{"name":"Bo","address":"bo@example.com"}},"receivedDateTime":"2026-07-16T09:00:00Z","isRead":true}
]}`

func TestMailList_Table(t *testing.T) {
	e := newEnv(t, jsonHandler(mailListBody))
	e.signIn("default")
	out, _, err := e.run("mail", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "ana@example.com")
	assert.Contains(t, out, "Invoice July")
	assert.Contains(t, out, "m2")
}

func TestMailList_JSONAndID(t *testing.T) {
	e := newEnv(t, jsonHandler(mailListBody))
	e.signIn("default")

	out, _, err := e.run("mail", "list", "-o", "json")
	require.NoError(t, err)
	var items []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &items))
	assert.Len(t, items, 2)

	out, _, err = e.run("mail", "list", "-o", "id")
	require.NoError(t, err)
	assert.Equal(t, "m1\nm2\n", out)
}

func TestMailList_CSV(t *testing.T) {
	e := newEnv(t, jsonHandler(mailListBody))
	e.signIn("default")
	out, _, err := e.run("mail", "list", "-o", "csv", "--columns", "id,subject")
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, "id,subject", lines[0])
	assert.Contains(t, lines[1], "m1")
}

func TestMailList_SearchGoesToGraph(t *testing.T) {
	var gotSearch string
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		gotSearch = r.URL.Query().Get("$search")
		_, _ = w.Write([]byte(`{"value":[]}`))
	})
	e.signIn("default")
	_, _, err := e.run("mail", "list", "--search", "invoice")
	require.NoError(t, err)
	assert.Equal(t, `"invoice"`, gotSearch)
}

func TestMailList_SearchFilterConflict(t *testing.T) {
	e := newEnv(t, jsonHandler(`{"value":[]}`))
	e.signIn("default")
	_, _, err := e.run("mail", "list", "--search", "a", "--filter", "b")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be combined")
}

func TestMailList_NotSignedIn(t *testing.T) {
	e := newEnv(t, jsonHandler(mailListBody))
	_, _, err := e.run("mail", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth login")
}

func TestMailGet_LetterView(t *testing.T) {
	e := newEnv(t, jsonHandler(`{
	  "id":"m1","subject":"Hello","receivedDateTime":"2026-07-17T10:00:00Z",
	  "from":{"emailAddress":{"name":"Ana","address":"ana@example.com"}},
	  "toRecipients":[{"emailAddress":{"name":"Juan","address":"juan@example.com"}}],
	  "hasAttachments":true,
	  "body":{"contentType":"text","content":"Hi Juan,\nsee attachment."}}`))
	e.signIn("default")
	out, _, err := e.run("mail", "get", "m1")
	require.NoError(t, err)
	assert.Contains(t, out, "From:     Ana <ana@example.com>")
	assert.Contains(t, out, "Subject:  Hello")
	assert.Contains(t, out, "Attachments: yes")
	assert.Contains(t, out, "see attachment.")
}

func TestMailGet_JSONReturnsFullResource(t *testing.T) {
	e := newEnv(t, jsonHandler(`{"id":"m1","subject":"Hello","body":{"contentType":"html","content":"<b>x</b>"}}`))
	e.signIn("default")
	out, _, err := e.run("mail", "get", "m1", "-o", "json")
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &m))
	assert.Equal(t, "m1", m["id"])
}

func TestCalendarEvents_DefaultWindow(t *testing.T) {
	var q map[string]string
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		q = map[string]string{
			"start":   r.URL.Query().Get("startDateTime"),
			"end":     r.URL.Query().Get("endDateTime"),
			"orderby": r.URL.Query().Get("$orderby"),
		}
		_, _ = w.Write([]byte(`{"value":[{"id":"e1","subject":"Standup","start":{"dateTime":"2026-07-20T09:00:00","timeZone":"UTC"},"end":{"dateTime":"2026-07-20T09:15:00","timeZone":"UTC"},"location":{"displayName":"Teams"}}]}`))
	})
	e.signIn("default")
	out, _, err := e.run("calendar", "events")
	require.NoError(t, err)
	assert.NotEmpty(t, q["start"], "default --from must be sent")
	assert.NotEmpty(t, q["end"], "default --to must be sent")
	assert.Equal(t, "start/dateTime", q["orderby"])
	assert.Contains(t, out, "Standup")
	assert.Contains(t, out, "Teams")
}

func TestCalendarEvents_BadDates(t *testing.T) {
	e := newEnv(t, jsonHandler(`{"value":[]}`))
	e.signIn("default")
	_, _, err := e.run("calendar", "events", "--from", "yesterday")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from")
}

func TestCalendarList(t *testing.T) {
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/me/events", r.URL.Path)
		_, _ = w.Write([]byte(`{"value":[{"id":"e1","subject":"1:1 series"}]}`))
	})
	e.signIn("default")
	out, _, err := e.run("calendar", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "1:1 series")
}

func TestCalendarAlias(t *testing.T) {
	e := newEnv(t, jsonHandler(`{"value":[]}`))
	e.signIn("default")
	_, _, err := e.run("cal", "list")
	require.NoError(t, err)
}

func TestMe(t *testing.T) {
	e := newEnv(t, jsonHandler(`{"displayName":"Juan","userPrincipalName":"juan@example.com","id":"u1"}`))
	e.signIn("default")
	out, _, err := e.run("me")
	require.NoError(t, err)
	assert.Contains(t, out, "juan@example.com")
}

func TestDryRun_PrintsCurlAndSendsNothing(t *testing.T) {
	e := newEnv(t, func(http.ResponseWriter, *http.Request) {
		t.Fatal("dry-run must not hit the server")
	})
	e.signIn("default")
	d := e.deps()
	var buf strings.Builder
	d.out = &buf // dry-run curls write here instead of os.Stdout
	out, _, err := runWithDeps(t, d, "mail", "list", "--dry-run")
	require.NoError(t, err)
	all := out + buf.String()
	assert.Contains(t, all, "curl -X GET")
	assert.Contains(t, all, "Bearer REDACTED")
	assert.NotContains(t, all, "fake-token")
}

func TestLimitAndAll_FollowNextLink(t *testing.T) {
	page := 0
	var srvURL string
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		page++
		if page < 3 {
			_, _ = w.Write([]byte(`{"value":[{"id":"p` + string(rune('0'+page)) + `"}],"@odata.nextLink":"` + srvURL + `/me/messages?page=next"}`))
			return
		}
		_, _ = w.Write([]byte(`{"value":[{"id":"last"}]}`))
	})
	srvURL = e.srv.URL
	e.signIn("default")
	out, _, err := e.run("mail", "list", "--all", "-o", "id")
	require.NoError(t, err)
	assert.Equal(t, 3, page)
	assert.Contains(t, out, "last")
}

func TestUnknownOutputFormatRejected(t *testing.T) {
	e := newEnv(t, jsonHandler(`{}`))
	_, _, err := e.run("me", "-o", "bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown output format")
}

func TestInvalidAccountNameRejected(t *testing.T) {
	e := newEnv(t, jsonHandler(`{}`))
	_, _, err := e.run("me", "-a", "../evil")
	require.Error(t, err)
}
