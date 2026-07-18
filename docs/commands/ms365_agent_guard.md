## ms365 agent guard

Generate agent-safety config that blocks destructive ms365 operations

### Synopsis

Classify every API command (read / write / irreversible) from the live command tree
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
accepted: a line that merely QUOTES a blocked command is denied.

```
ms365 agent guard --host <claude-code|codex|opencode> [flags]
```

### Examples

```
  ms365 agent guard --host claude-code
  ms365 agent guard --host claude-code --write          # write the files into .claude/
  ms365 agent guard --host codex --out ~/.codex/config.toml
  ms365 agent guard --host opencode --all-writes
```

### Options

```
      --all-writes    also hard-block ordinary writes, not just irreversible ops
  -h, --help          help for guard
      --host string   target agent host: claude-code|codex|opencode (required)
      --out string    write to this file instead of stdout
      --write         claude-code only: write hook + settings fragment under .claude/ (never overwrites)
```

### Options inherited from parent commands

```
  -a, --account string     named account (profile) to use
      --all                follow @odata.nextLink until exhausted (list commands)
      --base-url string    Microsoft Graph base URL override (sovereign clouds)
      --client-id string   Entra app (client) ID override — defaults to the embedded Microsoft Graph Command Line Tools app
      --columns strings    comma-separated columns to show
      --dry-run            print the equivalent curl and make no request
      --jq string          gojq expression applied to the response before rendering
      --limit int          max items to return across pages (list commands)
      --no-color           disable colored output
  -o, --output string      output format: table|json|yaml|csv|id
      --quiet              suppress non-essential chatter
      --show-token         reveal the bearer token in dry-run output
      --timezone string    return mail/calendar datetimes in this timezone (Prefer: outlook.timezone, e.g. "America/Caracas")
  -v, --verbose            verbose request logging (stderr)
```

### SEE ALSO

* [ms365 agent](ms365_agent.md)	 - AI-agent integration helpers

