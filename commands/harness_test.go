package commands

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jjuanrivvera/ms365-cli/internal/auth"
)

// fakeStore is an in-memory auth.Store so tests never touch a real OS keyring.
type fakeStore struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeStore() *fakeStore { return &fakeStore{data: map[string]string{}} }

func (f *fakeStore) Set(profile, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[profile] = token
	return nil
}

func (f *fakeStore) Get(profile string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.data[profile]; ok && t != "" {
		return t, nil
	}
	return "", auth.ErrNotFound
}

func (f *fakeStore) Delete(profile string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, profile)
	return nil
}

func (f *fakeStore) Backend() string { return "fake" }

// fakeProvider implements auth.Provider without MSAL: tests script its behavior per
// profile, mirroring how real profiles isolate sessions.
type fakeProvider struct {
	profile   string
	clientID  string
	scopes    []string
	state     *fakeAuthState
	loginErr  error
	silentErr error
}

// fakeAuthState is shared across provider instances (like the real keyring is).
type fakeAuthState struct {
	mu        sync.Mutex
	sessions  map[string]auth.Token // profile -> token
	loggedOut []string
}

func newFakeAuthState() *fakeAuthState { return &fakeAuthState{sessions: map[string]auth.Token{}} }

func (s *fakeAuthState) set(profile string, tok auth.Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[profile] = tok
}

func (p *fakeProvider) LoginDeviceCode(_ context.Context, prompt func(auth.DeviceCodePrompt)) (auth.Token, error) {
	if p.loginErr != nil {
		return auth.Token{}, p.loginErr
	}
	if prompt != nil {
		prompt(auth.DeviceCodePrompt{
			UserCode:        "ABC123",
			VerificationURL: "https://microsoft.com/devicelogin",
			Message:         "To sign in, open https://microsoft.com/devicelogin and enter the code ABC123.",
		})
	}
	tok := auth.Token{
		AccessToken: "fake-token-" + p.profile,
		ExpiresOn:   time.Now().Add(time.Hour),
		Scopes:      p.scopes,
		Username:    p.profile + "@example.com",
		TenantID:    "tenant-" + p.profile,
	}
	p.state.set(p.profile, tok)
	return tok, nil
}

func (p *fakeProvider) TokenSilent(context.Context) (auth.Token, error) {
	if p.silentErr != nil {
		return auth.Token{}, p.silentErr
	}
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	tok, ok := p.state.sessions[p.profile]
	if !ok {
		return auth.Token{}, auth.ErrNotLoggedIn
	}
	return tok, nil
}

func (p *fakeProvider) Logout(context.Context) error {
	p.state.mu.Lock()
	defer p.state.mu.Unlock()
	delete(p.state.sessions, p.profile)
	p.state.loggedOut = append(p.state.loggedOut, p.profile)
	return nil
}

// env wires one test invocation: an httptest Graph, an isolated config dir, a fake
// keyring, and a scripted auth provider.
type env struct {
	t     *testing.T
	srv   *httptest.Server
	store *fakeStore
	auth  *fakeAuthState
	// lastProvider records the clientID the tree resolved (for --client-id tests).
	lastClientID string
}

// newEnv starts a mock Graph server and isolates all state under t.TempDir().
func newEnv(t *testing.T, handler http.HandlerFunc) *env {
	t.Helper()
	e := &env{t: t, store: newFakeStore(), auth: newFakeAuthState()}
	if handler != nil {
		e.srv = httptest.NewServer(handler)
		t.Cleanup(e.srv.Close)
		t.Setenv("MS365_BASE_URL", e.srv.URL)
	} else {
		t.Setenv("MS365_BASE_URL", "")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MS365_ACCOUNT", "")
	t.Setenv("MS365_CLIENT_ID", "")
	t.Setenv("MS365_TIMEZONE", "")
	t.Setenv("NO_COLOR", "1")
	return e
}

// signIn pre-seeds a live session for a profile (as if auth login already ran).
func (e *env) signIn(profile string) {
	e.auth.set(profile, auth.Token{
		AccessToken: "fake-token-" + profile,
		ExpiresOn:   time.Now().Add(time.Hour),
		Scopes:      auth.DefaultScopes,
		Username:    profile + "@example.com",
		TenantID:    "tenant-" + profile,
	})
}

func (e *env) deps() *deps {
	d := newDeps()
	d.store = func() auth.Store { return e.store }
	d.provider = func(clientID, _, profile string, scopes []string) auth.Provider {
		e.lastClientID = clientID
		return &fakeProvider{profile: profile, clientID: clientID, scopes: scopes, state: e.auth}
	}
	return d
}

// run executes the real command tree with captured output.
func (e *env) run(args ...string) (string, string, error) {
	e.t.Helper()
	return runWithDeps(e.t, e.deps(), args...)
}

// runWithDeps builds a fresh tree from d and runs it with captured output.
func runWithDeps(t *testing.T, d *deps, args ...string) (string, string, error) {
	t.Helper()
	root := newRootCmd(d)
	root.SetArgs(args)
	var out, errB bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errB)
	err := root.ExecuteContext(t.Context())
	return out.String(), errB.String(), err
}

// jsonHandler answers every request with one body.
func jsonHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}
