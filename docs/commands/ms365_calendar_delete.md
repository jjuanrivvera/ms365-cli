## ms365 calendar delete

Delete a calendar event (Calendars.ReadWrite)

### Synopsis

DELETE /me/events/{id}. Attendees receive a cancellation when you organize the
event. Irreversible — the command asks for confirmation unless --yes is passed.

Deleting events needs the Calendars.ReadWrite delegated scope — grant it once per
account with:
  ms365 auth login -a <account> --scopes Calendars.ReadWrite

```
ms365 calendar delete <event-id> [flags]
```

### Examples

```
  ms365 calendar delete AAMkAGI2… --yes
  ms365 calendar list --filter "subject eq 'Old sync'" -o id | xargs -I{} ms365 calendar delete {} --yes
```

### Options

```
  -h, --help   help for delete
      --yes    skip the confirmation prompt
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

