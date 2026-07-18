## ms365 update

Update ms365 to the latest GitHub release

### Synopsis

Download the latest ms365 release, verify it against checksums.txt, and replace
the running binary in place. Use 'ms365 update check' to see what's available without
installing.

```
ms365 update [flags]
```

### Examples

```
  ms365 update
  ms365 update check
```

### Options

```
  -h, --help   help for update
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
* [ms365 update check](ms365_update_check.md)	 - Check for a newer release without installing it

