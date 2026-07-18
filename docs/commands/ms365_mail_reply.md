## ms365 mail reply

Reply to a message (Mail.Send)

### Synopsis

Reply to an existing message via /me/messages/{id}/reply — Exchange quotes the
original thread server-side and addresses the sender for you. --all replies to
every recipient (/replyAll).

Replying needs the Mail.Send delegated scope, which the default sign-in does NOT
request — grant it once per account with:
  ms365 auth login -a <account> --scopes Mail.Send

```
ms365 mail reply <message-id> [flags]
```

### Examples

```
  ms365 mail reply AAMkAGI2… --body "Works for me, see you then."
  ms365 mail reply AAMkAGI2… --all --body-file answer.txt
  ms365 mail list --top 1 -o id | xargs -I{} ms365 mail reply {} --body "ack"
```

### Options

```
      --all                reply to all recipients (/replyAll) instead of just the sender
      --body string        reply body text
      --body-file string   read the reply body from this file ("-" for stdin)
  -h, --help               help for reply
```

### Options inherited from parent commands

```
  -a, --account string     named account (profile) to use
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

