## ms365 auth logout

Sign out of the selected account

### Synopsis

Remove the selected account's cached tokens from the keyring. Other accounts are untouched.

```
ms365 auth logout [flags]
```

### Examples

```
  ms365 auth logout -a personal
  ms365 auth logout
```

### Options

```
  -h, --help   help for logout
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

