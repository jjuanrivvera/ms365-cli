## ms365 calendar events

Show calendar events in a time window (recurring events expanded)

### Synopsis

Query /me/calendarView between --from and --to (default: the next 7 days).
Unlike 'calendar list', this expands recurring events into their occurrences — it
answers "what is actually on my calendar".

Dates accept YYYY-MM-DD or full RFC 3339 timestamps. Combine with --timezone to get
start/end in your zone instead of UTC.

```
ms365 calendar events [flags]
```

### Examples

```
  ms365 calendar events
  ms365 calendar events --from 2026-07-20 --to 2026-07-27
  ms365 calendar events --timezone "America/Caracas" -o json
  ms365 calendar events -a work --limit 50
```

### Options

```
      --from string     window start (YYYY-MM-DD or RFC 3339; default now)
  -h, --help            help for events
      --select string   override the $select field list
      --to string       window end (YYYY-MM-DD or RFC 3339; default now+7d)
      --top int         page size requested from Graph ($top)
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

