package auth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// fakeAAD routes MSAL's HTTP calls by URL, standing in for login.microsoftonline.com so
// the real device-code and silent flows run end to end without network.
type fakeAAD struct {
	t          *testing.T
	tokenCalls int
}

const (
	fakeTenant   = "test-tenant-id"
	fakeUsername = "juan@example.com"
)

func (f *fakeAAD) CloseIdleConnections() {}

func (f *fakeAAD) Do(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	respond := func(body string) (*http.Response, error) {
		h := http.Header{}
		h.Set("Content-Type", "application/json; charset=utf-8")
		return &http.Response{StatusCode: 200, Header: h, Body: io.NopCloser(bytes.NewReader([]byte(body)))}, nil
	}
	authority := "https://login.microsoftonline.com/common"
	switch {
	case strings.Contains(url, "discovery/instance"):
		return respond(fmt.Sprintf(`{"tenant_discovery_endpoint":"%s/v2.0/.well-known/openid-configuration","api-version":"1.1","metadata":[{"preferred_network":"login.microsoftonline.com","preferred_cache":"login.microsoftonline.com","aliases":["login.microsoftonline.com"]}]}`, authority))
	case strings.Contains(url, "openid-configuration"):
		return respond(fmt.Sprintf(`{"token_endpoint":"%s/oauth2/v2.0/token","authorization_endpoint":"%s/oauth2/v2.0/authorize","device_authorization_endpoint":"%s/oauth2/v2.0/devicecode","issuer":"%s/v2.0","jwks_uri":"%s/discovery/v2.0/keys"}`, authority, authority, authority, authority, authority))
	case strings.Contains(url, "devicecode"):
		return respond(`{"device_code":"dev-code","expires_in":900,"interval":0,"message":"To sign in, use a web browser to open https://microsoft.com/devicelogin and enter the code ABC123.","user_code":"ABC123","verification_uri":"https://microsoft.com/devicelogin"}`)
	case strings.Contains(url, "/token"):
		f.tokenCalls++
		now := time.Now().Unix()
		payload := fmt.Sprintf(`{"aud":"client-id","exp":%d,"iat":%d,"iss":"https://login.microsoftonline.com/%s/v2.0","tid":"%s","preferred_username":"%s","oid":"oid-1","sub":"sub-1"}`,
			now+3600, now, fakeTenant, fakeTenant, fakeUsername)
		idToken := "header." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".signature"
		clientInfo := base64.RawURLEncoding.EncodeToString([]byte(`{"uid":"oid-1","utid":"` + fakeTenant + `"}`))
		return respond(fmt.Sprintf(`{"access_token":"fake-access-token","expires_in":3600,"expires_on":%d,"token_type":"Bearer","scope":"User.Read Mail.Read Calendars.Read","refresh_token":"fake-rt","id_token":"%s","client_info":"%s"}`,
			now+3600, idToken, clientInfo))
	default:
		f.t.Fatalf("fakeAAD: unexpected request %s", url)
		return nil, nil
	}
}

func newTestProvider(t *testing.T, profile string) (*MSALProvider, *fakeAAD) {
	t.Helper()
	keyring.MockInit()
	fake := &fakeAAD{t: t}
	p := NewMSALProvider("", "", profile, nil, New(t.TempDir()))
	p.HTTPClient = fake
	return p, fake
}

func TestNewMSALProvider_Defaults(t *testing.T) {
	p := NewMSALProvider("", "", "default", nil, nil)
	assert.Equal(t, DefaultClientID, p.ClientID)
	assert.Equal(t, DefaultAuthority, p.Authority)
	assert.Equal(t, DefaultScopes, p.Scopes)
}

func TestMSAL_DeviceCodeLogin_SilentAndLogout(t *testing.T) {
	p, _ := newTestProvider(t, "personal")

	var prompt DeviceCodePrompt
	tok, err := p.LoginDeviceCode(t.Context(), func(dp DeviceCodePrompt) { prompt = dp })
	require.NoError(t, err)

	assert.Equal(t, "ABC123", prompt.UserCode)
	assert.Equal(t, "https://microsoft.com/devicelogin", prompt.VerificationURL)
	assert.Contains(t, prompt.Message, "ABC123")

	assert.Equal(t, "fake-access-token", tok.AccessToken)
	assert.Equal(t, fakeUsername, tok.Username)
	assert.Equal(t, fakeTenant, tok.TenantID)
	assert.True(t, tok.ExpiresOn.After(time.Now()))

	// The cache accessor must have persisted the MSAL cache under the profile key.
	_, err = p.Store.Get(ProfileKey("personal"))
	require.NoError(t, err, "login must persist the MSAL cache for the profile")

	// A brand-new provider over the same store serves silently from the cache.
	p2 := NewMSALProvider("", "", "personal", nil, p.Store)
	p2.HTTPClient = p.HTTPClient
	silent, err := p2.TokenSilent(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "fake-access-token", silent.AccessToken)
	assert.Equal(t, fakeUsername, silent.Username)

	// Logout wipes the account and keyring entry; silent then fails with ErrNotLoggedIn.
	require.NoError(t, p2.Logout(t.Context()))
	_, err = p2.TokenSilent(t.Context())
	assert.ErrorIs(t, err, ErrNotLoggedIn)
}

func TestMSAL_TokenSilent_NotLoggedIn(t *testing.T) {
	p, _ := newTestProvider(t, "fresh")
	_, err := p.TokenSilent(t.Context())
	assert.ErrorIs(t, err, ErrNotLoggedIn)
}

func TestMSAL_ProfilesDoNotCross(t *testing.T) {
	p, _ := newTestProvider(t, "personal")
	_, err := p.LoginDeviceCode(t.Context(), nil)
	require.NoError(t, err)

	// Same store, different profile: no session.
	other := NewMSALProvider("", "", "work", nil, p.Store)
	other.HTTPClient = p.HTTPClient
	_, err = other.TokenSilent(t.Context())
	assert.ErrorIs(t, err, ErrNotLoggedIn)
}
