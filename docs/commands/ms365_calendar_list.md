## ms365 calendar list

List event masters (/me/events)

### Synopsis

List the events in the default calendar — the event MASTERS: single events and
recurrence series, without occurrence expansion. Use 'calendar events' for the
expanded time-window view.

```
ms365 calendar list [flags]
```

### Examples

```
  ms365 calendar list --top 25
  ms365 calendar list --filter "isOrganizer eq true" -o json
```

### Options

```
      --filter string   OData filter ($filter)
  -h, --help            help for list
      --select string   override the $select field list
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

