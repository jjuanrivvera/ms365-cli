# Getting started

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/jjuanrivvera/ms365-cli/main/install.sh | sh
```

Or `brew install jjuanrivvera/ms365-cli/ms365-cli`, or
`go install github.com/jjuanrivvera/ms365-cli/cmd/ms365@latest`.

## Sign in

```sh
ms365 auth login -a personal
```

The CLI prints a verification URL and a code. Open the URL in any browser (any device),
enter the code, and finish the Microsoft sign-in. Tokens land in your OS keyring under
that account name; nothing is stored in plaintext.

Add a second account any time — sessions never mix:

```sh
ms365 auth login -a work
ms365 auth status
```

On headless Linux without a Secret Service, set `MS365_KEYRING_PASSWORD` so the encrypted
file fallback uses a real key.

### Org tenants (work/school)

ms365 signs in with the Microsoft Graph Command Line Tools first-party app. Some tenants
require admin consent for it; if login fails with a consent/approval error, ask your
tenant admin, or register your own app (public client + device code enabled) and use
`ms365 auth login --client-id <your-app-guid>`.

## Read mail

```sh
ms365 mail list --top 20
ms365 mail list --folder inbox --search "from:ana invoice"
ms365 mail list --filter "isRead eq false" -o json
ms365 mail get <message-id>          # readable text body
ms365 mail get <message-id> -o json  # full Graph resource
```

## Calendar

```sh
ms365 calendar events                          # next 7 days, recurrences expanded
ms365 calendar events --from 2026-07-20 --to 2026-07-27
ms365 calendar events --timezone "America/Caracas"
ms365 calendar list                            # event masters (/me/events)
```

## Everything else

The raw escape hatch reaches any Graph endpoint with your signed-in session:

```sh
ms365 api GET me/mailFolders
ms365 api GET me/drive/root/children
ms365 api GET me/messages -q '$top=3' -q '$select=subject'
```

`--dry-run` on any command prints the equivalent `curl` (token redacted).
