# Changelog

All notable changes to this project are documented in this file. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
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
