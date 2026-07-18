package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jjuanrivvera/ms365-cli/internal/config"
)

func init() {
	metaRegistrars = append(metaRegistrars, func(d *deps) *cobra.Command {
		cfgCmd := &cobra.Command{
			Use:   "config",
			Short: "Inspect and edit ms365 configuration",
			Long: `The config file holds only non-secret settings (account profiles, aliases,
overrides). Tokens live in the OS keyring — never here.`,
		}
		cfgCmd.AddCommand(
			newConfigPathCmd(),
			newConfigViewCmd(d),
			newConfigUseCmd(d),
			newConfigListProfilesCmd(d),
			newConfigSetCmd(d),
		)
		return cfgCmd
	})
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), p)
			return nil
		},
	}
}

func newConfigViewCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Show the resolved configuration",
		Long:  "Print the config as YAML. No secrets are stored in the config, so nothing needs redacting.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := d.loadConfig()
			if err != nil {
				return err
			}
			b, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(b)
			return err
		},
	}
}

func newConfigUseCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "use <account>",
		Short: "Set the default account for future invocations",
		Example: `  ms365 config use personal
  ms365 config use work`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.ValidateProfileName(args[0]); err != nil {
				return err
			}
			cfg, err := d.loadConfig()
			if err != nil {
				return err
			}
			cfg.CurrentProfile = args[0]
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "default account is now %q\n", args[0])
			return nil
		},
	}
}

func newConfigListProfilesCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:     "list-profiles",
		Aliases: []string{"list-accounts", "accounts"},
		Short:   "List configured accounts",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := d.loadConfig()
			if err != nil {
				return err
			}
			names := cfg.ProfileNames()
			sort.Strings(names)
			for _, n := range names {
				marker := " "
				if n == cfg.CurrentProfile {
					marker = "*"
				}
				prof, _ := cfg.Profile(n)
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s\t%s\n", marker, n, prof.Username)
			}
			return nil
		},
	}
}

// configSetKeys are the per-account keys `config set` accepts.
var configSetKeys = map[string]func(*config.Profile, string){
	"client_id": func(p *config.Profile, v string) { p.ClientID = v },
	"base_url":  func(p *config.Profile, v string) { p.BaseURL = v },
	"timezone":  func(p *config.Profile, v string) { p.Timezone = v },
}

func newConfigSetCmd(d *deps) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a per-account option (client_id, base_url, timezone)",
		Long: `Set a non-secret option on the ACTIVE account profile (-a/--account selects it).
Keys: client_id (own app registration), base_url (sovereign cloud), timezone
(Prefer: outlook.timezone for mail/calendar output).`,
		Example: `  ms365 config set timezone "America/Caracas" -a personal
  ms365 config set client_id 00000000-0000-0000-0000-000000000000 -a work`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			setter, ok := configSetKeys[args[0]]
			if !ok {
				return fmt.Errorf("unknown key %q (want client_id|base_url|timezone)", args[0])
			}
			if args[0] == "base_url" && args[1] != "" {
				if err := config.ValidateBaseURL(args[1]); err != nil {
					return err
				}
			}
			profileName, cfg, err := d.resolveProfile()
			if err != nil {
				return err
			}
			prof, _ := cfg.Profile(profileName)
			setter(&prof, args[1])
			if err := cfg.SetProfile(profileName, prof); err != nil {
				return err
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s = %q on account %q\n", args[0], args[1], profileName)
			return nil
		},
	}
}
