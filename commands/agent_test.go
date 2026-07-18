package commands

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

// TestEveryAPICommandIsAnnotated locks §3b hardening #4: every runnable command outside
// the local/meta groups must carry an MCP classification annotation. A hand-added command
// without one shows up here (it would classify destructive at guard time — safe — but the
// build should fail loudly instead of shipping an unclassified verb).
func TestEveryAPICommandIsAnnotated(t *testing.T) {
	root := NewRootCmd()
	var offenders []string
	var walk func(cmd *cobra.Command, path string)
	walk = func(cmd *cobra.Command, path string) {
		for _, child := range cmd.Commands() {
			p := strings.TrimSpace(path + " " + child.Name())
			if path == "" && (slices.Contains(localGroups, child.Name()) || child.Name() == "api") {
				continue
			}
			if child.Runnable() {
				if kindAnnotated(child.Annotations) == "" {
					offenders = append(offenders, p)
				}
			}
			walk(child, p)
		}
	}
	walk(root, "")
	assert.Empty(t, offenders, "unannotated API commands (annotate in the builder or add to localGroups)")
}

func kindAnnotated(ann map[string]string) string {
	for _, k := range []string{annReadOnly, annOpenWorld, annDestructive} {
		if ann[k] == "true" {
			return k
		}
	}
	return ""
}

// TestClassifyAPICommands locks the read/write/destructive split of the v2 surface:
// reads allow, reversible writes (reply, event create/update) ask, and the irreversible
// ops (mail send — no unsend; calendar delete) hard-block. The raw `api` escape hatch is
// gated separately by HTTP method.
func TestClassifyAPICommands(t *testing.T) {
	cls := classifyAPICommands(false)

	paths := func(cmds []apiCmdInfo) []string {
		var out []string
		for _, c := range cmds {
			out = append(out, c.Path)
		}
		return out
	}
	reads, writes, destr := paths(cls.Read), paths(cls.Write), paths(cls.Destructive)

	for _, want := range []string{"mail list", "mail get", "calendar events", "calendar list", "me"} {
		assert.Contains(t, reads, want, "must be read-only")
	}
	assert.ElementsMatch(t, []string{"mail reply", "calendar create", "calendar update"}, writes)
	assert.ElementsMatch(t, []string{"mail send", "calendar delete"}, destr)

	// A destructive path must never also classify as read (verb-name collision guard).
	for _, r := range reads {
		assert.NotContains(t, destr, r, "path classified both read and destructive")
	}
}

// TestClassify_AllWritesPromotes verifies --all-writes moves every ordinary write into the
// hard-block bucket alongside the always-destructive ops.
func TestClassify_AllWritesPromotes(t *testing.T) {
	cls := classifyAPICommands(true)
	assert.Empty(t, cls.Write)
	var destr []string
	for _, c := range cls.Destructive {
		destr = append(destr, c.Path)
	}
	assert.ElementsMatch(t,
		[]string{"mail send", "mail reply", "calendar create", "calendar update", "calendar delete"},
		destr)
}

// TestAliasCrossProduct locks §3b hardening #5: alias paths are enumerated.
func TestAliasCrossProduct(t *testing.T) {
	var events apiCmdInfo
	for _, c := range classifyTree() {
		if c.Path == "calendar events" {
			events = c
			break
		}
	}
	require.NotEmpty(t, events.Path, "calendar events not found in classification")
	assert.Contains(t, events.AllPaths(), "cal events")

	var list apiCmdInfo
	for _, c := range classifyTree() {
		if c.Path == "mail list" {
			list = c
			break
		}
	}
	require.NotEmpty(t, list.Path)
	assert.Contains(t, list.AllPaths(), "messages list")
}

// TestMCPExcludesSetupCommands locks the MCP tool surface: no setup/meta/secret command
// may be exposed as a tool, and no secret flag may reach a tool schema.
func TestMCPExcludesSetupCommands(t *testing.T) {
	for _, name := range []string{"agent", "auth", "config", "alias", "init", "doctor", "completion", "version", "api", "update", "mcp"} {
		assert.Contains(t, excludedMCPPaths, name)
	}
	for _, flag := range []string{"show-token", "account", "profile", "base-url", "client-id"} {
		assert.Contains(t, secretFlags, flag)
	}

	// Walk the real tree: every excluded subtree's leaves must report excluded, and the
	// API commands must not.
	root := NewRootCmd()
	var walk func(cmd *cobra.Command)
	excludedSeen, includedSeen := map[string]bool{}, map[string]bool{}
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if child.Runnable() {
				if mcpExcluded(child) {
					excludedSeen[child.CommandPath()] = true
				} else {
					includedSeen[child.CommandPath()] = true
				}
			}
			walk(child)
		}
	}
	walk(root)
	assert.True(t, excludedSeen["ms365 auth login"])
	assert.True(t, excludedSeen["ms365 api"])
	assert.True(t, excludedSeen["ms365 update"], "self-updater must not be an MCP tool")
	assert.True(t, includedSeen["ms365 mail list"])
	assert.True(t, includedSeen["ms365 calendar events"])
	assert.True(t, includedSeen["ms365 me"])
	// Write tools ARE exposed — the annotations (readOnlyHint=false, destructiveHint) let
	// the host gate them; exclusion is only for meta/setup surfaces.
	assert.True(t, includedSeen["ms365 mail send"])
	assert.True(t, includedSeen["ms365 mail reply"])
	assert.True(t, includedSeen["ms365 calendar create"])
	assert.True(t, includedSeen["ms365 calendar update"])
	assert.True(t, includedSeen["ms365 calendar delete"])
}

// TestGuardRenderers smoke-checks each host output for the load-bearing content.
func TestGuardRenderers(t *testing.T) {
	cls := classifyAPICommands(false)

	claude, err := renderClaudeCode(cls)
	require.NoError(t, err)
	assert.Contains(t, claude, `Bash(ms365 api DELETE:*)`)
	assert.Contains(t, claude, `Bash(ms365 api POST:*)`)
	assert.Contains(t, claude, `Bash(ms365 alias set:*)`)
	assert.Contains(t, claude, `Bash(ms365 mail list:*)`)
	assert.Contains(t, claude, `Bash(ms365 mail send:*)`, "irreversible send must be denied")
	assert.Contains(t, claude, `Bash(ms365 messages send:*)`, "alias path must be denied too")
	assert.Contains(t, claude, `Bash(ms365 calendar delete:*)`)
	assert.Contains(t, claude, `Bash(ms365 mail reply:*)`, "writes go in the ask bucket")
	assert.Contains(t, claude, "mcp__ms365__ms365_mail_list")
	assert.Contains(t, claude, "mcp__ms365__ms365_mail_send")
	assert.Contains(t, claude, "mcp__ms365__ms365_calendar_delete")
	assert.Contains(t, claude, "PreToolUse")
	assert.Contains(t, claude, "blocked_cmds=(")

	codex, err := renderCodex(cls)
	require.NoError(t, err)
	assert.Contains(t, codex, `approval_policy = "on-request"`)
	assert.Contains(t, codex, `sandbox_mode = "read-only"`)

	oc, err := renderOpenCode(cls)
	require.NoError(t, err)
	assert.Contains(t, oc, `"permission"`)
	assert.Contains(t, oc, `"bash"`)
	assert.Contains(t, oc, `"ms365 api DELETE*": "deny"`)
	assert.Contains(t, oc, `"ms365 alias set*": "deny"`)
	assert.Contains(t, oc, `"ms365 mail list*": "allow"`)
	assert.Contains(t, oc, `"ms365 mail send*": "deny"`)
	assert.Contains(t, oc, `"ms365 cal delete*": "deny"`)
	assert.Contains(t, oc, `"ms365 mail reply*": "ask"`)
	assert.Contains(t, oc, `"ms365 calendar create*": "ask"`)
}

// TestGuardCommand_HostsAndWrite runs the actual cobra command per host, including the
// --write materialization (which must refuse to overwrite).
func TestGuardCommand_HostsAndWrite(t *testing.T) {
	e := newEnv(t, nil)

	out, _, err := e.run("agent", "guard", "--host", "claude-code")
	require.NoError(t, err)
	assert.Contains(t, out, "blocked_cmds=(")

	out, _, err = e.run("agent", "guard", "--host", "codex")
	require.NoError(t, err)
	assert.Contains(t, out, "sandbox_mode")

	out, _, err = e.run("agent", "guard", "--host", "opencode")
	require.NoError(t, err)
	assert.Contains(t, out, "permission")

	_, _, err = e.run("agent", "guard", "--host", "nope")
	require.Error(t, err)

	// --out writes a file.
	dest := filepath.Join(t.TempDir(), "guard.json")
	_, _, err = e.run("agent", "guard", "--host", "opencode", "--out", dest)
	require.NoError(t, err)
	b, err := os.ReadFile(dest) // #nosec G304 -- test temp path
	require.NoError(t, err)
	assert.Contains(t, string(b), "permission")

	// --write materializes under .claude/ and never overwrites.
	wd, err := os.Getwd()
	require.NoError(t, err)
	tmp := t.TempDir()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(wd) })

	_, _, err = e.run("agent", "guard", "--host", "claude-code", "--write")
	require.NoError(t, err)
	hook, err := os.ReadFile(filepath.Join(tmp, ".claude", "hooks", "ms365-guard.sh")) // #nosec G304 -- test temp path
	require.NoError(t, err)
	assert.Contains(t, string(hook), "blocked_cmds=(")
	_, _, err = e.run("agent", "guard", "--host", "claude-code", "--write")
	require.Error(t, err, "refuses to overwrite existing files")
}
