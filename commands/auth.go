package commands

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/ms365-cli/internal/auth"
	"github.com/jjuanrivvera/ms365-cli/internal/config"
)

func init() {
	metaRegistrars = append(metaRegistrars, func(d *deps) *cobra.Command {
		authCmd := &cobra.Command{
			Use:   "auth",
			Short: "Sign in and out of Microsoft 365 accounts",
			Long: `Authenticate via the OAuth device-code flow: the CLI prints a code and a URL, you
finish sign-in in any browser, and MSAL refresh tokens land in your OS keyring —
per account, so 'ms365 -a personal' and 'ms365 -a work' never share a session.`,
		}
		authCmd.AddCommand(newAuthLoginCmd(d), newAuthLogoutCmd(d), newAuthStatusCmd(d))
		return authCmd
	})
}

func newAuthLoginCmd(d *deps) *cobra.Command {
	var extraScopes []string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in with the device-code flow",
		Long: `Start a device-code sign-in for the selected account profile (-a/--account,
default "default"). The requested delegated scopes are User.Read, Mail.Read, and
Calendars.Read; add more with --scopes.

Works with personal Microsoft accounts (Outlook.com) and work/school (Entra) accounts.
Some org tenants restrict consent — if sign-in fails with an admin-consent error, ask a
tenant admin or use your own app registration via --client-id.`,
		Example: `  ms365 auth login -a personal
  ms365 auth login -a work --scopes Mail.ReadWrite
  ms365 auth login --client-id 00000000-0000-0000-0000-000000000000`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			profileName, cfg, err := d.resolveProfile()
			if err != nil {
				return err
			}
			scopes := append(append([]string{}, auth.DefaultScopes...), extraScopes...)
			clientID := d.resolveClientID(cfg, profileName)
			provider := d.authProviderWithScopes(clientID, profileName, scopes)

			fmt.Fprintf(cmd.ErrOrStderr(), "Signing in account %q…\n", profileName)
			tok, err := provider.LoginDeviceCode(cmd.Context(), func(p auth.DeviceCodePrompt) {
				out := cmd.OutOrStdout()
				if p.Message != "" {
					fmt.Fprintln(out, p.Message)
					return
				}
				fmt.Fprintf(out, "To sign in, open %s and enter the code %s\n", p.VerificationURL, p.UserCode)
			})
			if err != nil {
				return err
			}

			prof, _ := cfg.Profile(profileName)
			prof.Username = tok.Username
			prof.TenantID = tok.TenantID
			if clientID != auth.DefaultClientID {
				prof.ClientID = clientID
			}
			if err := cfg.SetProfile(profileName, prof); err != nil {
				return err
			}
			if cfg.CurrentProfile == "" {
				cfg.CurrentProfile = profileName
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Signed in as %s (tenant %s) on account %q.\n",
				tok.Username, describeTenant(tok.TenantID), profileName)
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&extraScopes, "scopes", nil, "additional delegated scopes to request (comma-separated)")
	return cmd
}

// authProviderWithScopes is authProvider with a custom scope set (login only).
func (d *deps) authProviderWithScopes(clientID, profile string, scopes []string) auth.Provider {
	if d.provider != nil {
		return d.provider(clientID, auth.DefaultAuthority, profile, scopes)
	}
	return auth.NewMSALProvider(clientID, auth.DefaultAuthority, profile, scopes, d.store())
}

func newAuthLogoutCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Sign out of the selected account",
		Long:  "Remove the selected account's cached tokens from the keyring. Other accounts are untouched.",
		Example: `  ms365 auth logout -a personal
  ms365 auth logout`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			profileName, cfg, err := d.resolveProfile()
			if err != nil {
				return err
			}
			provider := d.authProvider(d.resolveClientID(cfg, profileName), profileName)
			if err := provider.Logout(cmd.Context()); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Signed out account %q.\n", profileName)
			return nil
		},
	}
}

func newAuthStatusCmd(d *deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"whoami"},
		Short:   "Show signed-in identity, tenant, scopes, and token expiry",
		Long: `With -a/--account, show that account's session in detail. Without it, list every
configured account and whether it holds a live session.`,
		Example: `  ms365 auth status -a personal
  ms365 auth status`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := d.loadConfig()
			if err != nil {
				return err
			}
			if d.gf.account != "" {
				return d.printAccountStatus(cmd, cfg, d.gf.account)
			}
			names := cfg.ProfileNames()
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No accounts configured. Run `ms365 auth login -a <name>` to sign in.")
				return nil
			}
			sort.Strings(names)
			for _, n := range names {
				if err := d.printAccountStatus(cmd, cfg, n); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "%s: error: %v\n", n, err)
				}
			}
			return nil
		},
	}
	return cmd
}

// printAccountStatus resolves a silent token for one profile and prints its identity line.
func (d *deps) printAccountStatus(cmd *cobra.Command, cfg *config.Config, name string) error {
	if err := config.ValidateProfileName(name); err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	provider := d.authProvider(d.resolveClientID(cfg, name), name)
	tok, err := provider.TokenSilent(cmd.Context())
	if err != nil {
		if errors.Is(err, auth.ErrNotLoggedIn) {
			fmt.Fprintf(out, "%s: not signed in — run `ms365 auth login -a %s`\n", name, name)
			return nil
		}
		return err
	}
	marker := ""
	if cfg.CurrentProfile == name {
		marker = " (current)"
	}
	fmt.Fprintf(out, "%s%s:\n", name, marker)
	fmt.Fprintf(out, "  user:    %s\n", tok.Username)
	fmt.Fprintf(out, "  tenant:  %s\n", describeTenant(tok.TenantID))
	fmt.Fprintf(out, "  scopes:  %s\n", strings.Join(tok.Scopes, " "))
	fmt.Fprintf(out, "  expires: %s (%s)\n", tok.ExpiresOn.Local().Format(time.RFC3339), time.Until(tok.ExpiresOn).Round(time.Minute))
	return nil
}

// consumerTenantID is the fixed tenant behind every personal Microsoft account.
const consumerTenantID = "9188040d-6c67-4c5b-b112-36a304b66dad"

func describeTenant(id string) string {
	if id == consumerTenantID {
		return id + " (personal Microsoft account)"
	}
	return id
}
