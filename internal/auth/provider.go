package auth

import (
	"context"
	"errors"
	"time"
)

// DefaultClientID is the Microsoft Graph Command Line Tools first-party public client
// (the app behind Connect-MgGraph). It supports the device-code flow, is pre-registered in
// Entra tenants, and carries pre-consented delegated Graph scopes — the best odds of
// working in a restrictive org tenant without a custom app registration. Users who want
// their own registration override it with --client-id / MS365_CLIENT_ID / config.
const DefaultClientID = "14d82eec-204b-4c2f-b7e8-296a70dab67e"

// DefaultAuthority signs in both org (work/school) and personal Microsoft accounts.
const DefaultAuthority = "https://login.microsoftonline.com/common"

// DefaultScopes are the delegated scopes requested at login. Extend this list as the CLI
// grows write surfaces (Mail.Send, Calendars.ReadWrite, Files.Read, …).
var DefaultScopes = []string{"User.Read", "Mail.Read", "Calendars.Read"}

// ErrNotLoggedIn means no cached account exists for the profile — the caller should run
// `ms365 auth login -a <account>`.
var ErrNotLoggedIn = errors.New("not logged in")

// Token is one acquired access token plus the identity metadata auth status displays.
type Token struct {
	AccessToken string
	ExpiresOn   time.Time
	Scopes      []string
	Username    string // UPN, e.g. juan@contoso.com
	TenantID    string // Entra tenant (realm); "9188…" is the consumer (MSA) tenant
}

// DeviceCodePrompt carries the code the user must enter at the verification URL.
type DeviceCodePrompt struct {
	UserCode        string
	VerificationURL string
	Message         string
}

// Provider abstracts the MSAL public-client flows so commands are unit-testable without
// real Entra traffic.
type Provider interface {
	// LoginDeviceCode starts the device-code flow, calls prompt once the code is issued,
	// then blocks until the user completes sign-in (or ctx cancels).
	LoginDeviceCode(ctx context.Context, prompt func(DeviceCodePrompt)) (Token, error)
	// TokenSilent returns a valid access token from the cache, refreshing via MSAL when
	// needed. Returns ErrNotLoggedIn when the profile has no account.
	TokenSilent(ctx context.Context) (Token, error)
	// Logout removes the profile's cached account and tokens.
	Logout(ctx context.Context) error
}

// ProfileKey is the keyring entry name for a profile's serialized MSAL token cache
// (service "ms365-cli", key "profile-<name>").
func ProfileKey(profile string) string { return "profile-" + profile }
