package commands

import (
	"github.com/spf13/cobra"
)

// meColumns are the default table columns for the profile view.
var meColumns = []string{"displayName", "userPrincipalName", "mail", "jobTitle", "id"}

func init() {
	registrars = append(registrars, func(d *deps) *cobra.Command {
		cmd := &cobra.Command{
			Use:   "me",
			Short: "Show the signed-in user's profile (User.Read)",
			Long:  "Fetch /me — the whoami of Microsoft Graph. Useful to verify a session works.",
			Example: `  ms365 me
  ms365 me -a work -o json`,
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				c, _, err := d.getAPIClient()
				if err != nil {
					return err
				}
				body, err := c.Me(cmd.Context())
				if err != nil {
					return err
				}
				if d.gf.dryRun {
					return nil
				}
				return d.render(cmd, body, meColumns)
			},
		}
		return annotate(cmd, kindRead)
	})
}
