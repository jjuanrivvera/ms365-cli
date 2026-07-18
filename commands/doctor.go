package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/ms365-cli/internal/auth"
	"github.com/jjuanrivvera/ms365-cli/internal/config"
	"github.com/jjuanrivvera/ms365-cli/internal/version"
)

// doctorCheck is one diagnostic result.
type doctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func init() {
	metaRegistrars = append(metaRegistrars, func(d *deps) *cobra.Command {
		var jsonOut bool
		cmd := &cobra.Command{
			Use:   "doctor",
			Short: "Diagnose configuration, keyring, and Graph connectivity",
			Long: `Run local and remote health checks for the active account: config file, token
cache backend, cached session, and a live GET /me. Exits non-zero when any check
fails, so it is scriptable.`,
			Example: `  ms365 doctor
  ms365 doctor -a work --json`,
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				checks := d.runDoctor(cmd)
				failed := false
				for _, c := range checks {
					if !c.OK {
						failed = true
					}
				}
				if jsonOut {
					b, err := json.MarshalIndent(checks, "", "  ")
					if err != nil {
						return err
					}
					fmt.Fprintln(cmd.OutOrStdout(), string(b))
				} else {
					for _, c := range checks {
						mark := "✓"
						if !c.OK {
							mark = "✗"
						}
						fmt.Fprintf(cmd.OutOrStdout(), "%s %-14s %s\n", mark, c.Name, c.Detail)
					}
				}
				if failed {
					return fmt.Errorf("doctor found problems")
				}
				return nil
			},
		}
		cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
		return cmd
	})
}

func (d *deps) runDoctor(cmd *cobra.Command) []doctorCheck {
	var checks []doctorCheck
	add := func(name string, ok bool, detail string) {
		checks = append(checks, doctorCheck{Name: name, OK: ok, Detail: detail})
	}

	add("version", true, version.String())

	cfgPath, err := config.Path()
	if err != nil {
		add("config", false, err.Error())
		return checks
	}
	cfg, err := d.loadConfig()
	if err != nil {
		add("config", false, err.Error())
		return checks
	}
	add("config", true, cfgPath)

	profileName := cfg.ResolveProfileName(d.gf.account)
	add("account", true, profileName)

	store := d.store()
	add("keyring", true, "backend: "+store.Backend())

	provider := d.authProvider(d.resolveClientID(cfg, profileName), profileName)
	tok, err := provider.TokenSilent(cmd.Context())
	if err != nil {
		if errors.Is(err, auth.ErrNotLoggedIn) {
			add("session", false, fmt.Sprintf("not signed in — run `ms365 auth login -a %s`", profileName))
		} else {
			add("session", false, err.Error())
		}
		return checks
	}
	add("session", true, fmt.Sprintf("%s (expires %s)", tok.Username, tok.ExpiresOn.Local().Format("15:04 MST")))

	c, _, err := d.getAPIClient()
	if err != nil {
		add("graph", false, err.Error())
		return checks
	}
	if _, err := c.Me(cmd.Context()); err != nil {
		add("graph", false, err.Error())
	} else {
		add("graph", true, "GET /me OK ("+c.BaseURL()+")")
	}
	return checks
}
