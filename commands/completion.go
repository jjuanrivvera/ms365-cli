package commands

import (
	"github.com/spf13/cobra"
)

func init() {
	metaRegistrars = append(metaRegistrars, func(_ *deps) *cobra.Command {
		// Cobra generates a `completion` command automatically, but making it explicit lets
		// us document it and keeps dod-check/spec tooling honest about the surface.
		return &cobra.Command{
			Use:                   "completion [bash|zsh|fish|powershell]",
			Short:                 "Generate shell completion scripts",
			DisableFlagsInUseLine: true,
			ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
			Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
			Example: `  ms365 completion zsh > "${fpath[1]}/_ms365"
  ms365 completion bash > /etc/bash_completion.d/ms365
  ms365 completion fish > ~/.config/fish/completions/ms365.fish`,
			RunE: func(cmd *cobra.Command, args []string) error {
				root := cmd.Root()
				switch args[0] {
				case "bash":
					return root.GenBashCompletionV2(cmd.OutOrStdout(), true)
				case "zsh":
					return root.GenZshCompletion(cmd.OutOrStdout())
				case "fish":
					return root.GenFishCompletion(cmd.OutOrStdout(), true)
				default:
					return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
				}
			},
		}
	})
}
