package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestHookScript_BashExecution exercises the generated hook script with real bash to verify
// the adversarial cases: obfuscation, path-invoked binaries, the raw api escape hatch, and
// the benign lookalikes that must stay allowed. The shipped ms365 surface is read-only, so
// the blocked Bash set is `alias set` plus the api METHOD gate — exactly what the cases
// cover. Gated on a POSIX shell being available so it is safe in the regular suite.
func TestHookScript_BashExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash hook tests require a POSIX shell; skipping on windows")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found in PATH; skipping hook execution tests")
	}

	hookContent := hookScript(classifyAPICommands(false))
	tmpDir := t.TempDir()
	hookFile := filepath.Join(tmpDir, "ms365-guard.sh")
	if err := os.WriteFile(hookFile, []byte(hookContent), 0o755); err != nil { // #nosec G306 -- hook must be executable
		t.Fatalf("write hook: %v", err)
	}

	bashPayload := func(command string) string {
		b, _ := json.Marshal(map[string]any{
			"tool_name":  "Bash",
			"tool_input": map[string]any{"command": command},
		})
		return string(b)
	}
	mcpPayload := func(toolName string) string {
		b, _ := json.Marshal(map[string]any{
			"tool_name":  toolName,
			"tool_input": map[string]any{},
		})
		return string(b)
	}

	runHook := func(t *testing.T, payload string) string {
		t.Helper()
		cmd := exec.Command(bash, hookFile)
		cmd.Stdin = strings.NewReader(payload)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		// The hook always exits 0; the decision is in the JSON output.
		if err := cmd.Run(); err != nil {
			t.Logf("hook output: %s", out.String())
			t.Fatalf("hook script exited non-zero: %v", err)
		}
		return out.String()
	}

	isDenied := func(output string) bool {
		return strings.Contains(output, `"permissionDecision":"deny"`)
	}

	cases := []struct {
		name       string
		payload    string
		wantDenied bool
	}{
		// --- alias minting (the one blocked CLI path on a read-only surface) ---
		{"alias_set_denied", bashPayload(`ms365 alias set kill "api DELETE me/messages/1"`), true},
		// --- obfuscation ---
		{"quote_split_denied", bashPayload(`ms365 alias s""et kill "x"`), true},
		{"single_quote_split_denied", bashPayload(`ms365 alias s''et kill "x"`), true},
		{"backslash_denied", bashPayload(`ms365 alias s\et kill "x"`), true},
		{"newline_continuation_denied", bashPayload("ms365 alias \\\nset kill x"), true},
		// --- command position after separators ---
		{"after_semicolon_denied", bashPayload("true; ms365 alias set kill x"), true},
		{"after_pipe_denied", bashPayload("echo hi | ms365 alias set kill x"), true},
		{"after_and_denied", bashPayload("true && ms365 alias set kill x"), true},
		{"trailing_separator_denied", bashPayload("ms365 alias set kill x;true"), true},
		{"env_prefix_denied", bashPayload("env MS365_ACCOUNT=w ms365 alias set kill x"), true},
		// --- path-invoked binaries ---
		{"relative_path_binary_denied", bashPayload("./bin/ms365 alias set kill x"), true},
		{"absolute_path_binary_denied", bashPayload("/usr/local/bin/ms365 alias set kill x"), true},
		{"absolute_path_api_denied", bashPayload("/usr/local/bin/ms365 api DELETE me/messages/m1"), true},
		// --- raw api escape hatch (METHOD position; only GET/HEAD/OPTIONS pass) ---
		{"api_delete_denied", bashPayload("ms365 api DELETE me/messages/m1"), true},
		{"api_lowercase_delete_denied", bashPayload("ms365 api delete me/messages/m1"), true},
		{"api_post_denied", bashPayload("ms365 api POST me/sendMail -d '{}'"), true},
		{"api_patch_denied", bashPayload("ms365 api PATCH me/messages/m1 -d '{}'"), true},
		{"api_put_denied", bashPayload("ms365 api PUT me/mailFolders/x -d '{}'"), true},
		{"api_flag_before_method_denied", bashPayload("ms365 api -q x=1 DELETE me/messages/m1"), true},
		{"api_compound_get_then_delete_denied", bashPayload("ms365 api GET me;ms365 api DELETE me/messages/m1"), true},
		// --- raw api reads stay allowed ---
		{"api_get_allowed", bashPayload("ms365 api GET me"), false},
		{"api_get_lowercase_allowed", bashPayload("ms365 api get me"), false},
		{"api_head_allowed", bashPayload("ms365 api HEAD me"), false},
		{"api_get_delete_in_path_allowed", bashPayload("ms365 api GET me/mailFolders/deleteditems/messages"), false},
		// --- benign lookalikes that must stay allowed ---
		{"mail_list_allowed", bashPayload("ms365 mail list --top 20"), false},
		{"search_with_delete_in_arg_allowed", bashPayload(`ms365 mail list --search "how to delete an account"`), false},
		{"alias_list_allowed", bashPayload("ms365 alias list"), false},
		{"alias_remove_allowed", bashPayload("ms365 alias remove inbox"), false},
		{"quoted_blocked_cmd_in_arg_denied_conservatively", bashPayload(`rg "ms365 alias set" docs/`), true},
		{"cat_file_allowed", bashPayload("cat mail_delete.go"), false},
		{"other_binary_allowed", bashPayload("myms365 alias set kill x"), false},
		{"other_binary_api_allowed", bashPayload("myms365 api DELETE me/messages/m1"), false},
		{"cal_alias_events_allowed", bashPayload("ms365 cal events"), false},
		// --- MCP branch: read tools allowed; near-misses stay allowed ---
		{"mcp_mail_list_allowed", mcpPayload("mcp__ms365__ms365_mail_list"), false},
		{"mcp_calendar_events_allowed", mcpPayload("mcp__ms365__ms365_calendar_events"), false},
		{"mcp_near_miss_allowed", mcpPayload("mcp__ms365__ms365_mail_list2"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := runHook(t, tc.payload)
			if denied := isDenied(output); denied != tc.wantDenied {
				t.Errorf("want denied=%v, got denied=%v\noutput: %s", tc.wantDenied, denied, output)
			}
		})
	}
}

// TestHookScript_BashExecutionNoJq exercises the no-jq fallback path with a STRICT PATH: a
// bin dir holding only the POSIX tools the hook needs, so jq is genuinely unreachable
// (merely prepending an empty dir leaves jq resolvable — the test flaw that masked a
// fail-open bug in two audited repos, GOAL.md §3b #3).
func TestHookScript_BashExecutionNoJq(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash hook tests require a POSIX shell; skipping on windows")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found in PATH; skipping hook execution tests")
	}

	hookContent := hookScript(classifyAPICommands(false))
	tmpDir := t.TempDir()
	hookFile := filepath.Join(tmpDir, "ms365-guard.sh")
	if err := os.WriteFile(hookFile, []byte(hookContent), 0o755); err != nil { // #nosec G306 -- hook must be executable
		t.Fatalf("write hook: %v", err)
	}

	binDir := filepath.Join(tmpDir, "nojq-bin")
	if err := os.Mkdir(binDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, tool := range []string{"cat", "tr", "grep", "sed", "printf", "env"} {
		p, lerr := exec.LookPath(tool)
		if lerr != nil {
			continue // shell builtins (printf) need no symlink
		}
		if serr := os.Symlink(p, filepath.Join(binDir, tool)); serr != nil {
			t.Fatalf("symlink %s: %v", tool, serr)
		}
	}

	bashPayload := func(command string) string {
		b, _ := json.Marshal(map[string]any{
			"tool_name":  "Bash",
			"tool_input": map[string]any{"command": command},
		})
		return string(b)
	}

	runHookNoJq := func(t *testing.T, payload string) string {
		t.Helper()
		cmd := exec.Command(bash, hookFile)
		cmd.Stdin = strings.NewReader(payload)
		env := make([]string, 0, len(os.Environ()))
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "PATH=") {
				env = append(env, e)
			}
		}
		cmd.Env = append(env, "PATH="+binDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			t.Logf("hook output: %s", out.String())
			t.Fatalf("hook script exited non-zero: %v", err)
		}
		return out.String()
	}

	isDenied := func(output string) bool {
		return strings.Contains(output, `"permissionDecision":"deny"`)
	}

	cases := []struct {
		name       string
		payload    string
		wantDenied bool
	}{
		{"nojq_alias_set_denied", bashPayload("ms365 alias set kill x"), true},
		{"nojq_obfuscated_alias_set_denied", bashPayload(`ms365 alias s""et kill x`), true},
		{"nojq_path_binary_denied", bashPayload("./bin/ms365 alias set kill x"), true},
		{"nojq_api_delete_denied", bashPayload("ms365 api DELETE me/messages/m1"), true},
		{"nojq_cat_file_allowed", bashPayload("cat mail_delete.go"), false},
		{"nojq_mail_list_allowed", bashPayload("ms365 mail list --top 5"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output := runHookNoJq(t, tc.payload)
			if denied := isDenied(output); denied != tc.wantDenied {
				t.Errorf("want denied=%v, got denied=%v\noutput: %s", tc.wantDenied, denied, output)
			}
		})
	}
}
