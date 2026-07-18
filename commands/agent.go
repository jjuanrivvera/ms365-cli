package commands

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// apiCmdInfo records one API-backed command for the MCP server and agent guard to
// classify. It is derived from the LIVE command tree's annotations, so it stays correct as
// commands are added or renamed.
type apiCmdInfo struct {
	Path string // canonical CLI path, e.g. "mail list"
	Kind cmdKind
	// aliasPaths are the alternate CLI paths reachable through cobra aliases — every
	// ancestor-alias × leaf-alias combination except the canonical path (e.g. "messages
	// list", "cal events"). The guard must block these too, or an alias invocation silently
	// bypasses deny rules that only name the canonical path (§3b hardening #5).
	aliasPaths []string
}

// AllPaths returns every CLI path that invokes this command.
func (a apiCmdInfo) AllPaths() []string {
	return append([]string{a.Path}, a.aliasPaths...)
}

// localGroups are command subtrees that never talk to Microsoft Graph (setup/meta/local).
// Everything else must carry an annotation or classification fails closed as destructive —
// the "annotation gap" escape (§3b hardening #4).
var localGroups = []string{"auth", "config", "alias", "init", "doctor", "completion", "version", "update", "agent", "mcp", "help"}

// classifyTree walks a fresh command tree and buckets every API-backed leaf by its MCP
// annotation. The `api` escape hatch is excluded here — it is gated separately by HTTP
// method in the hook (§3b #6). Unannotated leaves classify as DESTRUCTIVE, never allowed.
func classifyTree() []apiCmdInfo {
	var out []apiCmdInfo
	var walk func(cmd *cobra.Command, names [][]string)
	walk = func(cmd *cobra.Command, names [][]string) {
		for _, child := range cmd.Commands() {
			seg := append([][]string{}, names...)
			seg = append(seg, append([]string{child.Name()}, child.Aliases...))
			if len(names) == 0 && (slices.Contains(localGroups, child.Name()) || child.Name() == "api") {
				continue
			}
			if child.Runnable() {
				canonical, aliases := pathsFromSegments(seg)
				out = append(out, apiCmdInfo{Path: canonical, Kind: kindOf(child), aliasPaths: aliases})
			}
			walk(child, seg)
		}
	}
	walk(NewRootCmd(), nil)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// pathsFromSegments expands [[calendar cal] [events]] into the canonical path and
// the alias cross-product.
func pathsFromSegments(segs [][]string) (string, []string) {
	combos := []string{""}
	for _, names := range segs {
		var next []string
		for _, prefix := range combos {
			for _, n := range names {
				p := n
				if prefix != "" {
					p = prefix + " " + n
				}
				next = append(next, p)
			}
		}
		combos = next
	}
	canonical := combos[0]
	var aliases []string
	aliases = append(aliases, combos[1:]...)
	return canonical, aliases
}

// kindOf reads the MCP annotation stamped by the generic builder. No annotation ⇒
// destructive: a hand-added command someone forgot to classify must fail closed, not fall
// through as harmless (§3b hardening #4).
func kindOf(cmd *cobra.Command) cmdKind {
	switch {
	case cmd.Annotations[annReadOnly] == "true":
		return kindRead
	case cmd.Annotations[annDestructive] == "true":
		return kindDestructive
	case cmd.Annotations[annOpenWorld] == "true":
		return kindWrite
	default:
		return kindDestructive
	}
}

// classification buckets every API command by safety level.
type classification struct {
	Read        []apiCmdInfo
	Write       []apiCmdInfo
	Destructive []apiCmdInfo
}

// classifyAPICommands splits the live tree into read/write/destructive. When allWrites is
// true, ordinary writes are promoted to the hard-block bucket.
func classifyAPICommands(allWrites bool) classification {
	var c classification
	for _, cmd := range classifyTree() {
		switch cmd.Kind {
		case kindRead:
			c.Read = append(c.Read, cmd)
		case kindWrite:
			if allWrites {
				c.Destructive = append(c.Destructive, cmd)
			} else {
				c.Write = append(c.Write, cmd)
			}
		case kindDestructive:
			c.Destructive = append(c.Destructive, cmd)
		}
	}
	return c
}

func init() {
	metaRegistrars = append(metaRegistrars, func(_ *deps) *cobra.Command {
		agentCmd := &cobra.Command{
			Use:   "agent",
			Short: "AI-agent integration helpers",
			Long:  "Generate safety configuration for AI agents that drive ms365.",
		}

		var host, out string
		var allWrites, write bool
		guard := &cobra.Command{
			Use:   "guard --host <claude-code|codex|opencode>",
			Short: "Generate agent-safety config that blocks destructive ms365 operations",
			Long: `Classify every API command (read / write / irreversible) from the live command tree
and emit host safety config: irreversible operations are hard-blocked, ordinary
writes require approval, and reads are allowed. The shipped ms365 surface is
read-only, so today the rails mainly gate the raw "api" escape hatch and "alias
set" — but the guard derives from the LIVE tree, so any future write/destructive
command is covered automatically. Cobra alias paths are covered too — "cal events"
hits the same rails as "calendar events".

For claude-code the output also includes a PreToolUse hook script
(.claude/hooks/ms365-guard.sh): it strips quote/backslash obfuscation, matches blocked
subcommand paths at the command position even for path-invoked binaries (./bin/ms365,
/usr/local/bin/ms365), and gates the raw "ms365 api <METHOD> <PATH>" escape hatch at
the METHOD position — only GET/HEAD/OPTIONS pass; POST/PUT/PATCH/DELETE are denied
case-insensitively, while a GET whose path merely contains "delete" stays allowed.
"ms365 alias set" is denied so an agent cannot mint a new shorthand for a blocked
command.

MCP-only operation is the hard guarantee; the Bash rails are best-effort — the hook
defeats quoting tricks and path prefixes, but not variable indirection
(a=DELETE; ms365 api $a x) or shell aliases. Conservative false positives are
accepted: a line that merely QUOTES a blocked command is denied.`,
			Example: `  ms365 agent guard --host claude-code
  ms365 agent guard --host claude-code --write          # write the files into .claude/
  ms365 agent guard --host codex --out ~/.codex/config.toml
  ms365 agent guard --host opencode --all-writes`,
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				cls := classifyAPICommands(allWrites)
				var content string
				var err error
				switch host {
				case "claude-code", "claude":
					if write {
						return writeClaudeCodeFiles(cmd, cls)
					}
					content, err = renderClaudeCode(cls)
				case "codex":
					content, err = renderCodex(cls)
				case "opencode":
					content, err = renderOpenCode(cls)
				default:
					return fmt.Errorf("unknown --host %q (want claude-code|codex|opencode)", host)
				}
				if err != nil {
					return err
				}
				if out != "" {
					if err := os.WriteFile(out, []byte(content), 0o600); err != nil {
						return err
					}
					fmt.Fprintf(cmd.ErrOrStderr(), "wrote %s safety config to %s\n", host, out)
					return nil
				}
				fmt.Fprint(cmd.OutOrStdout(), content)
				return nil
			},
		}
		guard.Flags().StringVar(&host, "host", "", "target agent host: claude-code|codex|opencode (required)")
		guard.Flags().StringVar(&out, "out", "", "write to this file instead of stdout")
		guard.Flags().BoolVar(&allWrites, "all-writes", false, "also hard-block ordinary writes, not just irreversible ops")
		guard.Flags().BoolVar(&write, "write", false, "claude-code only: write hook + settings fragment under .claude/ (never overwrites)")
		_ = guard.MarkFlagRequired("host")

		agentCmd.AddCommand(guard)
		return agentCmd
	})
}

// writeClaudeCodeFiles materializes the hook and settings fragment under .claude/,
// refusing to overwrite existing files (the user may have local edits).
func writeClaudeCodeFiles(cmd *cobra.Command, cls classification) error {
	hook := hookScript(cls)
	settings, err := claudeSettingsJSON(cls)
	if err != nil {
		return err
	}
	files := map[string]string{
		".claude/hooks/ms365-guard.sh":      hook,
		".claude/ms365-guard.settings.json": settings,
	}
	for path, content := range files {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists — remove it first or merge manually", path)
		}
		if err := os.MkdirAll(dirOf(path), 0o750); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(path, ".sh") {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "wrote %s\n", path)
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "merge .claude/ms365-guard.settings.json into .claude/settings.json to activate the hook")
	return nil
}

func dirOf(path string) string {
	if i := strings.LastIndex(path, "/"); i > 0 {
		return path[:i]
	}
	return "."
}

// bashPattern is the Claude-Code/OpenCode Bash permission pattern for a command path.
func bashPattern(path string) string { return "Bash(ms365 " + path + ":*)" }

// mcpToolPattern is the MCP tool name a host gates: ms365_<group>_<verb> under the ms365
// MCP server. These must be EXACT names — Claude permission rules are literal prefixes,
// not regex (§3b hardening #7). ophis joins path segments with "_" but keeps hyphens
// inside a segment, so only spaces are replaced here.
func mcpToolPattern(path string) string {
	return "mcp__ms365__ms365_" + strings.ReplaceAll(path, " ", "_")
}
