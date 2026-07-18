package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// MSALProvider implements Provider with a real MSAL public client (device-code flow, no
// client secret — the client ID is a public identifier, not a credential).
type MSALProvider struct {
	ClientID  string
	Authority string
	Profile   string
	Scopes    []string
	Store     Store

	// HTTPClient overrides MSAL's transport (tests point it at a fake Entra endpoint).
	// It matches MSAL's ops.HTTPClient structurally, so *http.Client satisfies it.
	HTTPClient HTTPClient
}

// HTTPClient mirrors MSAL's ops.HTTPClient so callers/tests never import MSAL internals.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// NewMSALProvider builds a provider for one profile with sane defaults.
func NewMSALProvider(clientID, authority, profile string, scopes []string, store Store) *MSALProvider {
	if clientID == "" {
		clientID = DefaultClientID
	}
	if authority == "" {
		authority = DefaultAuthority
	}
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}
	return &MSALProvider{ClientID: clientID, Authority: authority, Profile: profile, Scopes: scopes, Store: store}
}

// client builds the MSAL public client wired to this profile's persistent cache.
func (p *MSALProvider) client() (public.Client, error) {
	opts := []public.Option{
		public.WithAuthority(p.Authority),
		public.WithCache(&cacheAccessor{store: p.Store, profile: p.Profile}),
	}
	if p.HTTPClient != nil {
		opts = append(opts, public.WithHTTPClient(p.HTTPClient))
	}
	return public.New(p.ClientID, opts...)
}

// LoginDeviceCode runs the two-step device-code flow: surface the code, then poll until
// the user finishes signing in on their other device/browser.
func (p *MSALProvider) LoginDeviceCode(ctx context.Context, prompt func(DeviceCodePrompt)) (Token, error) {
	c, err := p.client()
	if err != nil {
		return Token{}, err
	}
	dc, err := c.AcquireTokenByDeviceCode(ctx, p.Scopes)
	if err != nil {
		return Token{}, fmt.Errorf("start device-code flow: %w", err)
	}
	if prompt != nil {
		prompt(DeviceCodePrompt{
			UserCode:        dc.Result.UserCode,
			VerificationURL: dc.Result.VerificationURL,
			Message:         dc.Result.Message,
		})
	}
	res, err := dc.AuthenticationResult(ctx)
	if err != nil {
		return Token{}, fmt.Errorf("device-code sign-in failed: %w", err)
	}
	return tokenFromResult(res), nil
}

// TokenSilent serves from the cache, letting MSAL redeem the refresh token when the access
// token is stale. ErrNotLoggedIn tells the caller to run auth login.
func (p *MSALProvider) TokenSilent(ctx context.Context) (Token, error) {
	c, err := p.client()
	if err != nil {
		return Token{}, err
	}
	accounts, err := c.Accounts(ctx)
	if err != nil {
		return Token{}, err
	}
	if len(accounts) == 0 {
		return Token{}, ErrNotLoggedIn
	}
	res, err := c.AcquireTokenSilent(ctx, p.Scopes, public.WithSilentAccount(accounts[0]))
	if err != nil {
		return Token{}, fmt.Errorf("%w — run `ms365 auth login -a %s` (silent refresh failed: %v)", ErrNotLoggedIn, p.Profile, err)
	}
	return tokenFromResult(res), nil
}

// Logout removes every cached account for the profile and deletes its keyring entry, so
// no refresh token survives.
func (p *MSALProvider) Logout(ctx context.Context) error {
	c, err := p.client()
	if err != nil {
		return err
	}
	accounts, err := c.Accounts(ctx)
	if err == nil {
		for _, a := range accounts {
			_ = c.RemoveAccount(ctx, a)
		}
	}
	return p.Store.Delete(ProfileKey(p.Profile))
}

func tokenFromResult(res public.AuthResult) Token {
	// The ID token's tid is the account's REAL tenant; Account.Realm echoes the authority
	// tenant, which is the literal "common" for multi-tenant sign-in.
	tenant := res.IDToken.TenantID
	if tenant == "" {
		tenant = res.Account.Realm
	}
	return Token{
		AccessToken: res.AccessToken,
		ExpiresOn:   res.ExpiresOn,
		Scopes:      res.GrantedScopes,
		Username:    res.Account.PreferredUsername,
		TenantID:    tenant,
	}
}
