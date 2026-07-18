## ms365 calendar create

Create a calendar event (Calendars.ReadWrite)

### Synopsis

Create an event in the default calendar via POST /me/events. --from/--to are
wall-clock times interpreted in --timezone (default UTC); attendees receive an
invitation. --online-meeting provisions a Teams meeting link.

Creating events needs the Calendars.ReadWrite delegated scope, which the default
sign-in does NOT request — grant it once per account with:
  ms365 auth login -a <account> --scopes Calendars.ReadWrite

```
ms365 calendar create [flags]
```

### Examples

```
  ms365 calendar create --subject "1:1 Ana" --from 2026-07-21T10:00 --to 2026-07-21T10:30
  ms365 calendar create --subject "Sprint review" --from 2026-07-24T15:00 --to 2026-07-24T16:00 \
    --timezone "America/Caracas" --attendee ana@x.com --attendee bo@x.com --online-meeting
  ms365 calendar create --subject "Focus block" --from 2026-07-22T09:00 --to 2026-07-22T11:00 --location "Home office"
```

### Options

```
      --attendee strings   attendee email (repeatable or comma-separated)
      --body string        event body text
      --from string        start (YYYY-MM-DDTHH:MM, wall-clock in --timezone)
  -h, --help               help for create
      --location string    location display name
      --online-meeting     provision a Teams online meeting
      --subject string     event subject
      --to string          end (YYYY-MM-DDTHH:MM, wall-clock in --timezone)
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

* [ms365 calendar](ms365_calendar.md)	 - Read your Outlook calendar (Calendars.Read)

