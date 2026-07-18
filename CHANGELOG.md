# Changelog

All notable changes to this project are documented in this file. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- **Write surface (v2)** — the default sign-in stays read-only; grant write scopes per
  account with `auth login --scopes …`:
  - `mail send` (`POST /me/sendMail`): `--to/--cc/--bcc` (repeatable or CSV),
    `--subject`, `--body`/`--body-file` (`-` for stdin), `--html`,
    `--save-to-sent` (default true). Needs `Mail.Send`.
  - `mail reply <message-id>` (`POST /me/messages/{id}/reply`, `--all` for `/replyAll`):
    `--body`/`--body-file`. Needs `Mail.Send`.
  - `calendar create` (`POST /me/events`): `--subject`, `--from/--to` (wall-clock in
    `--timezone`, default UTC), `--location`, `--body`, `--attendee` (repeatable),
    `--online-meeting` (Teams). Needs `Calendars.ReadWrite`.
  - `calendar update <event-id>` (`PATCH /me/events/{id}`): same flags, sends ONLY the
    flags you pass. Needs `Calendars.ReadWrite`.
  - `calendar delete <event-id>` (`DELETE /me/events/{id}`): asks for confirmation
    unless `--yes`. Needs `Calendars.ReadWrite`.
- Scope pre-check for write commands: a token signed in without the needed scope fails
  with the exact re-login command (`run: ms365 auth login -a <account> --scopes …`).
- `agent guard` / MCP annotations cover the write surface: `mail send` and
  `calendar delete` are hard-blocked as irreversible; `mail reply` and
  `calendar create/update` require approval.
- Device-code sign-in via MSAL (public client, embedded Microsoft Graph Command Line
  Tools client ID; `--client-id` override).
- Multi-account profiles (`-a/--account`) with per-profile MSAL token caches in the OS
  keyring and an AES-256-GCM encrypted-file fallback (`MS365_KEYRING_PASSWORD`).
- `auth login` / `auth logout` / `auth status` (whoami), per-account.
- `mail list` (folders, `$search`/`$filter`, `--top/--limit/--all`) and `mail get`
  (text body by default, `--raw` for HTML).
- `calendar events` (`/me/calendarView`, recurring events expanded, `--from/--to`
  defaulting to the next 7 days) and `calendar list` (`/me/events`).
- `me` — the Graph whoami.
- Meta commands: `config`, `init`, `doctor`, `completion`, `alias`, `api` (raw escape
  hatch), `version`, `update` (self-updater), `mcp` (MCP server), `agent guard`.
- Output formats: table (default), json, yaml, csv, id; `--columns`, `--jq`; CSV
  formula-injection guard; terminal-escape sanitization.
- `@odata.nextLink` pagination with `--all`/`--limit`, Retry-After-honoring retries,
  `--dry-run` curl with token redaction, `--timezone` (Prefer: outlook.timezone).
