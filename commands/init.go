package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/ms365-cli/internal/config"
)

func init() {
	metaRegistrars = append(metaRegistrars, func(d *deps) *cobra.Command {
		cmd := &cobra.Command{
			Use:     "init",
			Aliases: []string{"setup"},
			Short:   "First-run setup: name an account and sign in",
			Long: `Interactive first-run wizard: pick an account name (e.g. "personal" or "work"),
then complete a device-code sign-in. Equivalent to 'ms365 auth login -a <name>'.`,
			Example: `  ms365 init`,
			Args:    cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				name := d.gf.account
				if name == "" {
					entered, err := promptLine(cmd, `Account name (how you'll refer to this sign-in, e.g. "personal") [default]: `)
					if err != nil {
						return err
					}
					if entered == "" {
						entered = config.DefaultProfile
					}
					name = entered
				}
				if err := config.ValidateProfileName(name); err != nil {
					return err
				}
				d.gf.account = name

				login := newAuthLoginCmd(d)
				login.SetOut(cmd.OutOrStdout())
				login.SetErr(cmd.ErrOrStderr())
				login.SetContext(cmd.Context())
				if err := login.RunE(login, nil); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\nAll set. Try:\n  ms365 mail list -a %s\n  ms365 calendar events -a %s\n", name, name)
				return nil
			},
		}
		return cmd
	})
}
