## ms365 api

Send a raw authenticated Graph request (escape hatch)

### Synopsis

Call any Microsoft Graph endpoint directly. The path is relative to the configured
base URL (default https://graph.microsoft.com/v1.0), e.g. me/drive/root/children.

This is the documented escape hatch for anything ms365 does not wrap as a
first-class command. It honors --dry-run, -o/--output, and --jq like every other
command. Non-GET methods are never auto-retried, and writes need scopes beyond the
default read-only set (grant them at login with --scopes).

```
ms365 api <METHOD> <PATH> [-d body] [-q key=value ...] [flags]
```

### Examples

```
  ms365 api GET me/mailFolders
  ms365 api GET me/messages -q '$top=5' -q '$select=subject'
  ms365 api GET me/drive/root/children
  ms365 api POST me/sendMail -d @mail.json --dry-run
```

### Options

```
  -d, --data string         JSON body: inline, @file, or - for stdin
  -h, --help                help for api
  -q, --query stringArray   query parameter key=value (repeatable)
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

