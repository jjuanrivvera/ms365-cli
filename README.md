# ms365 — Microsoft 365 from your terminal

A fast, scriptable CLI for **Microsoft 365** via the **Microsoft Graph API**: read your
Outlook mail, browse your calendar, and inspect your profile — with named accounts so a
personal Outlook.com sign-in and a work/school tenant live side by side and never share
tokens.

- **Device-code sign-in** (MSAL, public client — no secrets): `ms365 auth login`, finish in
  any browser.
- **Multi-account** with `-a/--account`: `ms365 mail list -a personal`, `ms365 calendar
  events -a work`. Each account has its own token cache in your OS keyring.
- **Agent-ready**: an MCP server (`ms365 mcp`) exposes every command as an annotated tool,
  and `ms365 agent guard` generates safety rails for Claude Code / Codex / OpenCode.
- **Pipe-clean output**: table, JSON, YAML, CSV, `-o id`, plus a built-in `--jq`.

## Install

macOS / Linux (no package manager needed):

```sh
curl -fsSL https://raw.githubusercontent.com/jjuanrivvera/ms365-cli/main/install.sh | sh
```

Homebrew:

```sh
brew install jjuanrivvera/ms365-cli/ms365-cli
```

Scoop (Windows):

```powershell
scoop bucket add ms365-cli https://github.com/jjuanrivvera/scoop-ms365-cli
scoop install ms365-cli
```

Go:

```sh
go install github.com/jjuanrivvera/ms365-cli/cmd/ms365@latest
```

## Quickstart

```sh
# Sign in (device-code flow — the CLI prints a URL and a code)
ms365 auth login -a personal

# Who am I?
ms365 auth status
ms365 me

# Mail
ms365 mail list --top 20
ms365 mail list --folder inbox --search "invoice"
ms365 mail list --filter "isRead eq false" -o json
ms365 mail get <message-id>

# Calendar (next 7 days by default; recurring events expanded)
ms365 calendar events
ms365 calendar events --from 2026-07-20 --to 2026-07-27 --timezone "America/Caracas"

# A second account, side by side
ms365 auth login -a work
ms365 mail list -a work
```

Anything Graph exposes that isn't wrapped yet is one escape hatch away:

```sh
ms365 api GET me/mailFolders
ms365 api GET me/drive/root/children
```

## Authentication notes

- Sign-in uses the **OAuth device-code flow** with the Microsoft Graph Command Line Tools
  first-party client ID (the same app `Connect-MgGraph` uses). Requested delegated scopes:
  `User.Read Mail.Read Calendars.Read`.
- Tokens (MSAL cache incl. refresh tokens) live in the **OS keyring**, one entry per
  account. Headless Linux without a Secret Service falls back to an AES-256-GCM encrypted
  file — set `MS365_KEYRING_PASSWORD` for a real key.
- Org tenants can restrict user consent; if login fails with an admin-consent error, ask a
  tenant admin or use your own app registration: `ms365 auth login --client-id <guid>`.

## Output & scripting

Every command takes `-o table|json|yaml|csv|id`, `--columns`, `--jq`, `--quiet`,
`--no-color`. List commands take `--limit` and `--all` (auto-follows `@odata.nextLink`).
`--dry-run` prints the equivalent `curl` (token redacted) and sends nothing.

## AI agents

```sh
ms365 mcp claude          # install the MCP server into Claude Code
ms365 agent guard --host claude-code --write   # safety rails (blocks raw writes)
```

## Development

`make verify` is the gate: fmt, vet, lint, tests, ≥80% coverage, spec gates, and the
Definition-of-Done checks. See `AGENTS.md` for the contributor guide and `DECISIONS.md`
for pinned API assumptions.

## License

MIT
