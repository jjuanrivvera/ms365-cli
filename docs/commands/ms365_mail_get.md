## ms365 mail get

Show one message with its body as text

### Synopsis

Fetch a single message. By default the body is down-converted to plain text and
printed in a readable letter form; --raw keeps the original (usually HTML) body.
Use -o json for the complete Graph resource.

```
ms365 mail get <message-id> [flags]
```

### Examples

```
  ms365 mail get AAMkAGI2…
  ms365 mail get AAMkAGI2… --raw -o json
  ms365 mail list --top 1 -o id | xargs ms365 mail get
```

### Options

```
  -h, --help   help for get
      --raw    keep the original body (HTML) instead of text down-conversion
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

* [ms365 mail](ms365_mail.md)	 - Read Outlook mail (Mail.Read)

