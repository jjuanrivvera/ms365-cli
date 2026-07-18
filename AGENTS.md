# AGENTS.md — working in the ms365-cli repo

`ms365` is a command-line tool for **Microsoft 365** via the **Microsoft Graph API v1.0**
(delegated mail/calendar/identity surface — reads plus the v2 writes: send/reply mail,
create/update/delete events), built to the cliwright standard (Go + Cobra + GoReleaser).
This file orients an AI agent (or human) contributing.

## The one rule that matters
**`make verify` is the gate.** A change is done only when `make verify` exits `0`. It runs
`make check` (fmt, vet, golangci-lint, tests) + `spec-check` (the built surface matches
`api-manifest.json`) + `spec-completeness` (manifest vs the 16643-operation Graph v1.0
enumeration — passes via the recorded coverage-waiver in `DECISIONS.md`) + `cover-check`
(≥80% coverage) + `dod-check.sh`. Run the full `make verify` for any change that touches
the command surface or a documented behavior — not just `make check`.

## Architecture (where things live)
- `internal/api/` — the Graph client core: bearer auth via a `TokenFunc`, idempotent-only
  retry honoring `Retry-After` (delta-seconds + HTTP-date) with full-jitter backoff,
  `@odata.nextLink` pagination (`--all`/`--limit`, page cap, same-host guard), dry-run curl
  with token redaction, `APIError` with actionable hints, and typed service methods
  (`MailList`, `MailGet`, `MailSend`, `MailReply`, `CalendarView`, `EventsList`,
  `EventCreate`, `EventUpdate`, `EventDelete`, `Me`). Pattern B (service-layer) — see
  DECISIONS.md #6 for the trigger.
- `internal/auth/` — MSAL device-code auth behind the `Provider` interface (fakeable in
  tests): per-profile token caches serialized via a `cache.ExportReplace` accessor into the
  OS keyring (service `ms365-cli`, key `profile-<name>`), AES-256-GCM encrypted-file
  fallback (`MS365_KEYRING_PASSWORD`). The embedded client ID is the Microsoft Graph
  Command Line Tools first-party app (DECISIONS.md #1).
- `commands/` — the cobra tree. `init()` appends builders to `registrars`/`metaRegistrars`;
  `NewRootCmd(deps)` drains the queue onto a fresh root (wootctl-style hybrid — no mutable
  global root). MCP annotations are stamped via `annotate(cmd, kind)` as commands are built.
- `internal/{config,output,version,update}` — account profiles + manual precedence (no
  Viper), the table/json/yaml/csv/id renderer (CSV formula-injection guard, terminal-escape
  sanitizer, NO_COLOR), build metadata, the checksum-verified self-updater.
- `cmd/ms365/main.go` — `signal.NotifyContext` (Ctrl-C cancels in-flight work, including
  the device-code wait) + alias expansion before cobra parses.

## Graph specifics you must not re-derive
- Auth is the device-code flow against `https://login.microsoftonline.com/common`; scopes
  are the `auth.DefaultScopes` constant (`User.Read Mail.Read Calendars.Read`). NO client
  secret exists anywhere. **DefaultScopes stay read-only** — write commands need a
  per-account grant (`auth login --scopes Mail.Send` / `Calendars.ReadWrite`) and
  pre-check the cached token's grant with the exact re-login hint (DECISIONS.md #18).
- Profiles NEVER cross: `-a personal` and `-a work` each own a keyring entry.
- Pagination is `@odata.nextLink` (absolute URL — follow only on the same host).
- `$search` on messages is KQL, must be quoted, and cannot combine with `$filter`/`$orderby`
  (the CLI enforces this before the request). `ConsistencyLevel: eventual` is a
  directory-object requirement, NOT needed for mail — don't add it there.
- `/me/calendarView` requires `startDateTime`+`endDateTime` and expands recurrences;
  `/me/events` lists masters. Both are first-class commands.
- Throttling: 429 + `Retry-After` — honored with bounded retries; no quota headers exist.
- `Prefer: outlook.timezone="…"` drives datetime rendering (`--timezone`).

## House rules
- Comments explain **WHY**, not WHAT.
- Thread `cmd.Context()` everywhere; never `context.Background()` (it breaks Ctrl-C).
- Secrets live in the OS keyring — never in config, code, or commit messages.
- Pin every ambiguous API assumption in `DECISIONS.md`; read it back, never re-decide.
- Surface changes require updating `api-manifest.json` AND regenerating docs
  (`make docs-gen`) in the same commit.
- MCP exclusions are by EXACT path (`commands/mcp.go`), never substring.
- New commands ship with tests in the same commit — coverage is a ratchet (≥80%).
