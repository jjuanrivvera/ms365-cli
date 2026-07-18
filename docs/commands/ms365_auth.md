## ms365 auth

Sign in and out of Microsoft 365 accounts

### Synopsis

Authenticate via the OAuth device-code flow: the CLI prints a code and a URL, you
finish sign-in in any browser, and MSAL refresh tokens land in your OS keyring —
per account, so 'ms365 -a personal' and 'ms365 -a work' never share a session.

### Options

```
  -h, --help   help for auth
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

* [ms365](ms365.md)	 - A fast, scriptable CLI for Microsoft 365 (Microsoft Graph)
* [ms365 auth login](ms365_auth_login.md)	 - Sign in with the device-code flow
* [ms365 auth logout](ms365_auth_logout.md)	 - Sign out of the selected account
* [ms365 auth status](ms365_auth_status.md)	 - Show signed-in identity, tenant, scopes, and token expiry

