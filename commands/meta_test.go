package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jjuanrivvera/ms365-cli/internal/config"
	"github.com/jjuanrivvera/ms365-cli/internal/update"
	"github.com/jjuanrivvera/ms365-cli/internal/version"
)

func TestAuthLogin_DeviceCodeFlow(t *testing.T) {
	e := newEnv(t, nil)
	out, errOut, err := e.run("auth", "login", "-a", "personal")
	require.NoError(t, err)

	assert.Contains(t, out, "ABC123", "the device code must be shown")
	assert.Contains(t, out, "microsoft.com/devicelogin")
	assert.Contains(t, out, "Signed in as personal@example.com")
	assert.Contains(t, errOut, `Signing in account "personal"`)

	// The profile landed in config with identity metadata.
	cfg, err := config.Load()
	require.NoError(t, err)
	prof, ok := cfg.Profile("personal")
	require.True(t, ok)
	assert.Equal(t, "personal@example.com", prof.Username)
	assert.Equal(t, "tenant-personal", prof.TenantID)
	assert.Equal(t, "personal", cfg.CurrentProfile, "first login becomes the default account")
}

func TestAuthLogin_ProfilesAreIsolated(t *testing.T) {
	e := newEnv(t, nil)
	_, _, err := e.run("auth", "login", "-a", "personal")
	require.NoError(t, err)
	_, _, err = e.run("auth", "login", "-a", "work")
	require.NoError(t, err)

	assert.Contains(t, e.auth.sessions, "personal")
	assert.Contains(t, e.auth.sessions, "work")
	assert.NotEqual(t, e.auth.sessions["personal"].AccessToken, e.auth.sessions["work"].AccessToken)
}

func TestAuthLogin_CustomClientIDPersisted(t *testing.T) {
	e := newEnv(t, nil)
	_, _, err := e.run("auth", "login", "-a", "own", "--client-id", "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", e.lastClientID)
	cfg, _ := config.Load()
	prof, _ := cfg.Profile("own")
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", prof.ClientID)
}

func TestAuthLogout_RemovesOnlyThatProfile(t *testing.T) {
	e := newEnv(t, nil)
	e.signIn("personal")
	e.signIn("work")
	out, _, err := e.run("auth", "logout", "-a", "personal")
	require.NoError(t, err)
	assert.Contains(t, out, `Signed out account "personal"`)
	assert.NotContains(t, e.auth.sessions, "personal")
	assert.Contains(t, e.auth.sessions, "work", "logout must not touch other accounts")
}

func TestAuthStatus_SingleAccount(t *testing.T) {
	e := newEnv(t, nil)
	e.signIn("personal")
	out, _, err := e.run("auth", "status", "-a", "personal")
	require.NoError(t, err)
	assert.Contains(t, out, "personal@example.com")
	assert.Contains(t, out, "tenant-personal")
	assert.Contains(t, out, "User.Read")
	assert.Contains(t, out, "expires:")
}

func TestAuthStatus_ListsAllProfiles(t *testing.T) {
	e := newEnv(t, nil)
	_, _, err := e.run("auth", "login", "-a", "personal")
	require.NoError(t, err)
	_, _, err = e.run("auth", "login", "-a", "work")
	require.NoError(t, err)
	// Sign work out so the listing shows both states.
	_, _, err = e.run("auth", "logout", "-a", "work")
	require.NoError(t, err)

	out, _, err := e.run("auth", "status")
	require.NoError(t, err)
	assert.Contains(t, out, "personal (current)")
	assert.Contains(t, out, "work: not signed in")
}

func TestAuthStatus_NoAccounts(t *testing.T) {
	e := newEnv(t, nil)
	out, _, err := e.run("auth", "status")
	require.NoError(t, err)
	assert.Contains(t, out, "No accounts configured")
}

func TestWhoamiAlias(t *testing.T) {
	e := newEnv(t, nil)
	e.signIn("default")
	out, _, err := e.run("auth", "whoami", "-a", "default")
	require.NoError(t, err)
	assert.Contains(t, out, "default@example.com")
}

func TestInit_RunsLoginForNamedAccount(t *testing.T) {
	e := newEnv(t, nil)
	d := e.deps()
	root := newRootCmd(d)
	root.SetArgs([]string{"init"})
	root.SetIn(strings.NewReader("personal\n"))
	var out, errB strings.Builder
	root.SetOut(&out)
	root.SetErr(&errB)
	require.NoError(t, root.ExecuteContext(t.Context()))
	assert.Contains(t, out.String(), "Signed in as personal@example.com")
	assert.Contains(t, out.String(), "mail list -a personal")
}

func TestConfig_PathViewUseSetList(t *testing.T) {
	e := newEnv(t, nil)
	_, _, err := e.run("auth", "login", "-a", "personal")
	require.NoError(t, err)

	out, _, err := e.run("config", "path")
	require.NoError(t, err)
	assert.Contains(t, out, "config.yaml")

	out, _, err = e.run("config", "set", "timezone", "America/Caracas", "-a", "personal")
	require.NoError(t, err)
	assert.Contains(t, out, "timezone")

	out, _, err = e.run("config", "view")
	require.NoError(t, err)
	assert.Contains(t, out, "America/Caracas")
	assert.NotContains(t, out, "fake-token", "no secret may ever appear in config view")

	_, _, err = e.run("config", "use", "personal")
	require.NoError(t, err)

	out, _, err = e.run("config", "list-profiles")
	require.NoError(t, err)
	assert.Contains(t, out, "* personal")
	assert.Contains(t, out, "personal@example.com")
}

func TestConfigSet_Validation(t *testing.T) {
	e := newEnv(t, nil)
	_, _, err := e.run("config", "set", "bogus-key", "x")
	require.Error(t, err)
	_, _, err = e.run("config", "set", "base_url", "http://evil.example.com")
	require.Error(t, err, "cleartext non-loopback base_url must be rejected")
}

func TestDoctor_HealthyJSON(t *testing.T) {
	e := newEnv(t, jsonHandler(`{"displayName":"Juan"}`))
	e.signIn("default")
	out, _, err := e.run("doctor", "--json")
	require.NoError(t, err)
	var checks []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &checks))
	byName := map[string]bool{}
	for _, c := range checks {
		byName[c["name"].(string)] = c["ok"].(bool)
	}
	for _, name := range []string{"version", "config", "account", "keyring", "session", "graph"} {
		assert.True(t, byName[name], "check %q must pass", name)
	}
}

func TestDoctor_NotSignedInFails(t *testing.T) {
	e := newEnv(t, nil)
	out, _, err := e.run("doctor")
	require.Error(t, err)
	assert.Contains(t, out, "not signed in")
}

func TestAPI_GetRendersJSON(t *testing.T) {
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/me/mailFolders", r.URL.Path)
		assert.Equal(t, "5", r.URL.Query().Get("$top"))
		_, _ = w.Write([]byte(`{"value":[{"id":"f1","displayName":"Inbox"}]}`))
	})
	e.signIn("default")
	out, _, err := e.run("api", "GET", "me/mailFolders", "-q", "$top=5", "-o", "json")
	require.NoError(t, err)
	assert.Contains(t, out, "Inbox")
}

func TestAPI_InvalidMethodAndQuery(t *testing.T) {
	e := newEnv(t, jsonHandler(`{}`))
	e.signIn("default")
	_, _, err := e.run("api", "YOLO", "me")
	require.Error(t, err)
	_, _, err = e.run("api", "GET", "me", "-q", "no-equals")
	require.Error(t, err)
}

func TestAPI_EmptyBodyNote(t *testing.T) {
	e := newEnv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	})
	e.signIn("default")
	out, _, err := e.run("api", "DELETE", "me/messages/m1")
	require.NoError(t, err)
	assert.Contains(t, out, "HTTP 204")
}

func TestAlias_SetListRemoveAndExpansion(t *testing.T) {
	e := newEnv(t, jsonHandler(mailListBody))
	e.signIn("default")

	out, _, err := e.run("alias", "set", "inbox", "mail list --folder inbox")
	require.NoError(t, err)
	assert.Contains(t, out, "inbox")

	out, _, err = e.run("alias", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "inbox = mail list --folder inbox")

	expanded := ExpandAliases([]string{"inbox", "--top", "5"})
	assert.Equal(t, []string{"mail", "list", "--folder", "inbox", "--top", "5"}, expanded)

	// A built-in name can never be aliased or shadowed.
	_, _, err = e.run("alias", "set", "mail", "calendar list")
	require.Error(t, err)
	assert.Equal(t, []string{"mail"}, ExpandAliases([]string{"mail"}))

	out, _, err = e.run("alias", "remove", "inbox")
	require.NoError(t, err)
	assert.Contains(t, out, "removed")
	_, _, err = e.run("alias", "remove", "inbox")
	require.Error(t, err)
}

func TestVersion_PlainAndJSON(t *testing.T) {
	e := newEnv(t, nil)
	out, _, err := e.run("version")
	require.NoError(t, err)
	assert.Contains(t, out, "ms365")

	out, _, err = e.run("version", "--json")
	require.NoError(t, err)
	var info version.Info
	require.NoError(t, json.Unmarshal([]byte(out), &info))
	assert.NotEmpty(t, info.Version)
}

func TestUpdateCheck_ReportsNewer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"tag_name":"v99.0.0","assets":[]}`)
	}))
	defer srv.Close()
	old := newUpdater
	newUpdater = func() *update.Updater { return update.NewUpdaterWithBaseURL(version.Version, srv.URL) }
	defer func() { newUpdater = old }()

	e := newEnv(t, nil)
	out, _, err := e.run("update", "check")
	require.NoError(t, err)
	assert.Contains(t, out, "v99.0.0")
	assert.Contains(t, out, "development build")
}

func TestCompletion_AllShells(t *testing.T) {
	e := newEnv(t, nil)
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		out, _, err := e.run("completion", shell)
		require.NoError(t, err, shell)
		assert.NotEmpty(t, out, shell)
	}
	_, _, err := e.run("completion", "tcsh")
	require.Error(t, err)
}

func TestVersionCheck_AgainstFakeGitHub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.0.1"}`))
	}))
	defer srv.Close()
	old := latestReleaseURL
	latestReleaseURL = srv.URL
	defer func() { latestReleaseURL = old }()

	e := newEnv(t, nil)
	out, _, err := e.run("version", "--check")
	require.NoError(t, err)
	assert.Contains(t, out, "0.0.1")
}

func TestUpdateRun_DevBuildIsNoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v99.0.0","assets":[]}`))
	}))
	defer srv.Close()
	old := newUpdater
	newUpdater = func() *update.Updater { return update.NewUpdaterWithBaseURL(version.Version, srv.URL) }
	defer func() { newUpdater = old }()

	e := newEnv(t, nil)
	out, _, err := e.run("update")
	require.NoError(t, err)
	assert.Contains(t, out, "latest version")
}

func TestAPI_DataArgVariants(t *testing.T) {
	var gotBody string
	e := newEnv(t, func(w http.ResponseWriter, r *http.Request) {
		b := new(strings.Builder)
		_, _ = io.Copy(b, r.Body)
		gotBody = b.String()
		_, _ = w.Write([]byte(`{}`))
	})
	e.signIn("default")

	_, _, err := e.run("api", "POST", "me/sendMail", "-d", `{"inline":true}`)
	require.NoError(t, err)
	assert.Equal(t, `{"inline":true}`, gotBody)

	f := filepath.Join(t.TempDir(), "body.json")
	require.NoError(t, os.WriteFile(f, []byte(`{"file":true}`), 0o600))
	_, _, err = e.run("api", "POST", "me/sendMail", "-d", "@"+f)
	require.NoError(t, err)
	assert.Equal(t, `{"file":true}`, gotBody)

	d := e.deps()
	root := newRootCmd(d)
	root.SetArgs([]string{"api", "POST", "me/sendMail", "-d", "-"})
	root.SetIn(strings.NewReader(`{"stdin":true}`))
	var out, errB strings.Builder
	root.SetOut(&out)
	root.SetErr(&errB)
	require.NoError(t, root.ExecuteContext(t.Context()))
	assert.Equal(t, `{"stdin":true}`, gotBody)

	_, _, err = e.run("api", "POST", "me/sendMail", "-d", "@/nonexistent/nope.json")
	require.Error(t, err)
}
