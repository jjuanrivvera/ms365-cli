package commands

import (
	"slices"

	"github.com/njayp/ophis"
	"github.com/spf13/cobra"
)

// excludedMCPPaths are the EXACT command paths (relative to the root) kept out of the MCP
// tool surface: setup/meta commands an agent should not drive, the raw `api` escape hatch
// (it would bypass per-command annotations), and the `agent`/`mcp` subtrees so an agent can
// neither re-enter the server nor disable its own guardrails. Exact-path matching, never
// substring — a substring like "update" would also drop a real resource verb.
var excludedMCPPaths = []string{
	"agent", "auth", "config", "alias", "init", "doctor", "completion", "version", "api",
	"update", "mcp",
}

// secretFlags must never reach the MCP tool schema: an agent must not read the token,
// switch accounts, retarget the Graph endpoint, or swap the app registration. The server
// uses whatever account is active at startup.
var secretFlags = []string{"show-token", "account", "profile", "base-url", "client-id"}

// mcpExcluded reports whether cmd sits at or under an excluded top-level path.
func mcpExcluded(cmd *cobra.Command) bool {
	for c := cmd; c != nil && c.HasParent(); c = c.Parent() {
		if !c.Parent().HasParent() { // c is a top-level command
			return slices.Contains(excludedMCPPaths, c.Name())
		}
	}
	return false
}

func init() {
	metaRegistrars = append(metaRegistrars, func(_ *deps) *cobra.Command {
		// ophis walks the command tree and exposes each runnable leaf as an MCP tool,
		// replaying the cobra command on invocation so tools reuse the same client,
		// keyring, and account.
		return ophis.Command(&ophis.Config{
			ToolNamePrefix: "ms365",
			Selectors: []ophis.Selector{{
				CmdSelector:           func(cmd *cobra.Command) bool { return !mcpExcluded(cmd) },
				InheritedFlagSelector: ophis.ExcludeFlags(secretFlags...),
			}},
		})
	})
}
