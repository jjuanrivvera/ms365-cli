## ms365 auth login

Sign in with the device-code flow

### Synopsis

Start a device-code sign-in for the selected account profile (-a/--account,
default "default"). The requested delegated scopes are User.Read, Mail.Read, and
Calendars.Read; add more with --scopes.

Works with personal Microsoft accounts (Outlook.com) and work/school (Entra) accounts.
Some org tenants restrict consent — if sign-in fails with an admin-consent error, ask a
tenant admin or use your own app registration via --client-id.

```
ms365 auth login [flags]
```

### Examples

```
  ms365 auth login -a personal
  ms365 auth login -a work --scopes Mail.ReadWrite
  ms365 auth login --client-id 00000000-0000-0000-0000-000000000000
```

### Options

```
  -h, --help             help for login
      --scopes strings   additional delegated scopes to request (comma-separated)
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

* [ms365 auth](ms365_auth.md)	 - Sign in and out of Microsoft 365 accounts

