## ms365 mail send

Send an email (Mail.Send)

### Synopsis

Compose and send a message via /me/sendMail. The body is plain text by default;
--html marks it as HTML. --body-file reads the body from a file ("-" for stdin).

Sending needs the Mail.Send delegated scope, which the default sign-in does NOT
request — grant it once per account with:
  ms365 auth login -a <account> --scopes Mail.Send

```
ms365 mail send [flags]
```

### Examples

```
  ms365 mail send --to ana@example.com --subject "Lunch?" --body "12:30 at the usual place"
  ms365 mail send --to a@x.com,b@x.com --cc boss@x.com --subject "Report" --body-file report.txt
  ms365 mail send --to ops@x.com --subject "Alert" --html --body "<b>disk full</b>" --save-to-sent=false
```

### Options

```
      --bcc strings        BCC address (repeatable or comma-separated)
      --body string        message body text
      --body-file string   read the body from this file ("-" for stdin)
      --cc strings         CC address (repeatable or comma-separated)
  -h, --help               help for send
      --html               send the body as HTML instead of plain text
      --save-to-sent       keep a copy in Sent Items (default true)
      --subject string     message subject
      --to strings         recipient address (repeatable or comma-separated)
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

