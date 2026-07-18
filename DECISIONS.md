# DECISIONS.md — pinned assumptions for ms365-cli

Every ambiguous or judgment call is recorded here (question → decision → why) so iterations
never silently re-decide (cliwright GOAL.md §11).

1. **Client ID** → embed `14d82eec-204b-4c2f-b7e8-296a70dab67e` (Microsoft Graph Command
   Line Tools, the first-party app behind `Connect-MgGraph`). Verified against Microsoft
   Learn Q&A ("Microsoft Graph Command Line Tools Enterprise application",
   learn.microsoft.com/en-us/answers/questions/1619076) and multiple independent admin
   references (practical365.com, undocumented-features.com) — all attribute this exact GUID
   to that app. It is a public client (no secret), supports the device-code flow, and is
   pre-registered in Entra tenants with pre-consented delegated Graph scopes — the best odds
   in restrictive org tenants. Overridable via `--client-id` / `MS365_CLIENT_ID` / per-account
   `client_id` in config for users with their own app registration.
2. **Authority** → `https://login.microsoftonline.com/common`: signs in both work/school
   (Entra) and personal Microsoft accounts. Not per-profile-configurable in v1; a tenant-
   restricted user can still sign in via /common.
3. **Auth flow** → device-code only (MSAL for Go, `apps/public`). No client secret is ever
   involved (public client). No interactive-browser flow in v1 — device code works everywhere
   including headless boxes, and matches the requested scope.
4. **Profiles** → the multi-profile selector is `-a/--account` (a profile IS a Microsoft
   account); `--profile` is kept as a hidden alias per the cliwright standard. Env override:
   `MS365_ACCOUNT`. Each profile has its OWN MSAL token cache serialized through a
   `cache.ExportReplace` accessor into the OS keyring (service `ms365-cli`, key
   `profile-<name>`), with the fleet-standard AES-256-GCM encrypted-file fallback
   (`credentials.enc` in the config dir, key from `MS365_KEYRING_PASSWORD` via scrypt, else a
   host-bound obfuscation key). Profiles never share tokens.
5. **Scopes** → `auth.DefaultScopes = [User.Read, Mail.Read, Calendars.Read]` — a single
   constant designed to grow (Mail.Send, Calendars.ReadWrite …). `auth login --scopes`
   appends extra scopes for one profile's consent grant.
6. **Resource pattern** → Pattern B (service-layer) per the §11 trigger: the shipped Graph
   endpoints are not uniform CRUD-on-a-resource (`/me/calendarView` is a required-window
   time-series query, message bodies need `Prefer: outlook.body-content-type` handling,
   `$search` vs `$filter` are mutually exclusive on messages). Typed `MailList/MailGet/
   CalendarView/EventsList/Me` methods on one shared client core.
7. **Enumeration** → `api_method_total` = 16643 operations, counted from the official
   `microsoftgraph/msgraph-metadata` OpenAPI (`openapi/v1.0/openapi.yaml`,
   `grep -c 'operationId:'`, fetched 2026-07-18).
8. **coverage-waiver**: shipping the delegated read-only mail/calendar/identity surface
   first (5 wrapped operations of 16643 enumerated — Microsoft Graph spans every M365
   workload and full coverage is neither feasible nor desirable for a hand-ergonomic CLI).
   The `api <METHOD> <PATH>` escape hatch reaches the remaining endpoints; write surfaces
   (Mail.Send, event create) are deferred to v2 by design.
9. **Pagination** → `@odata.nextLink` auto-follow with `--all`/`--limit`, hard page cap 50,
   and a same-host guard (never follow a nextLink off the configured Graph host — a
   malicious/poisoned response must not exfiltrate the bearer token).
10. **$search semantics** → on `/me/messages`, `$search` (KQL, quoted) needs NO
    `ConsistencyLevel: eventual` header (that requirement applies to directory objects like
    /users). The CLI therefore doesn't send it for mail; the generic `List` accepts extra
    headers so directory-style resources can add it later. `--search` and `--filter` are
    rejected together — Graph itself errors on the combination.
11. **Timezone** → `--timezone`/`MS365_TIMEZONE`/per-account `timezone` sets
    `Prefer: outlook.timezone="…"` so mail/calendar datetimes render in the user's zone.
    Default: unset (Graph returns UTC).
12. **Throttling** → Graph exposes no request-quota headers on these endpoints; strategy is
    bounded retries (max 3) honoring `Retry-After` (delta-seconds and HTTP-date) with
    full-jitter backoff, idempotent methods only. No fixed-RPS limiter — a single-user
    read CLI cannot realistically sustain Graph's documented mailbox limits.
13. **`mail get` default view** → letter-style text rendering (headers + body down-converted
    via `Prefer: outlook.body-content-type="text"`); `--raw` keeps HTML; `-o json` returns
    the untouched Graph resource.
14. **MCP surface** → excluded by EXACT top-level path (`agent auth config alias init doctor
    completion version api update mcp`), never substring (fleet lesson: substring "update"
    also kills a real resource verb). Secret/instance flags (`show-token`, `account`,
    `profile`, `base-url`, `client-id`) never reach tool schemas.
15. **Conditional patterns (§3d)** → event-store: N/A (Graph has durable server-side mail/
    calendar history + search). Offline cache/sync: not in v1 (mail is not date-scoped
    time-series; deferred). Multi-credential groups: N/A (one bearer token). spec-sync: not
    wired (the 38 MB OpenAPI re-derivation is out of scope for v1). smoke.yml: N/A (no
    durable headless credential — device-code tokens expire). Typed-library adoption: MSAL
    adopted for auth; no mature Go Graph client wrap (msgraph-sdk-go is codegen-heavy and
    would fight the generic renderer). Self-update: shipped (fleet standard).
16. **CI/release** → thin callers of the fleet reusables (`jjuanrivvera/.github` go-ci /
    go-release) per the monorepo AGENTS.md — NOT cliwright's standalone templates, because
    this is a jjuanrivvera fleet repo.
17. **Keyring size limits** → Windows Credential Manager caps blobs (~2.5 KB) below a
    typical MSAL cache; `auth.Store.Set` transparently falls back to the encrypted file when
    the keyring write fails, so login still works there.
