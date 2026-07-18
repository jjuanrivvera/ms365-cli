## ms365 calendar update

Update a calendar event (Calendars.ReadWrite)

### Synopsis

PATCH /me/events/{id} with ONLY the flags you pass — unset flags leave the event
untouched. --attendee REPLACES the whole attendee list. Get event ids from
'calendar list -o id' or 'calendar events -o id'.

Updating events needs the Calendars.ReadWrite delegated scope — grant it once per
account with:
  ms365 auth login -a <account> --scopes Calendars.ReadWrite

```
ms365 calendar update <event-id> [flags]
```

### Examples

```
  ms365 calendar update AAMkAGI2… --subject "1:1 Ana (moved)"
  ms365 calendar update AAMkAGI2… --from 2026-07-21T11:00 --to 2026-07-21T11:30 --timezone "America/Caracas"
  ms365 calendar update AAMkAGI2… --location "Room 3" --online-meeting=false
```

### Options

```
      --attendee strings   attendee email (repeatable or comma-separated)
      --body string        event body text
      --from string        start (YYYY-MM-DDTHH:MM, wall-clock in --timezone)
  -h, --help               help for update
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

