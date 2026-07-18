## ms365 config set

Set a per-account option (client_id, base_url, timezone)

### Synopsis

Set a non-secret option on the ACTIVE account profile (-a/--account selects it).
Keys: client_id (own app registration), base_url (sovereign cloud), timezone
(Prefer: outlook.timezone for mail/calendar output).

```
ms365 config set <key> <value> [flags]
```

### Examples

```
  ms365 config set timezone "America/Caracas" -a personal
  ms365 config set client_id 00000000-0000-0000-0000-000000000000 -a work
```

### Options

```
  -h, --help   help for set
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

* [ms365 config](ms365_config.md)	 - Inspect and edit ms365 configuration

