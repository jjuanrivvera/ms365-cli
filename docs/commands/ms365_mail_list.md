## ms365 mail list

List messages

### Synopsis

List messages from the mailbox (newest first). --folder accepts a well-known name
(inbox, sentitems, drafts, deleteditems, junkemail, archive) or a folder id.

--search is Exchange KQL ($search); --filter is OData ($filter). Microsoft Graph
rejects combining them, and $search results ignore ordering.

```
ms365 mail list [flags]
```

### Examples

```
  ms365 mail list --top 20
  ms365 mail list --folder inbox --search "from:ana subject:invoice"
  ms365 mail list --filter "isRead eq false" -o json
  ms365 mail list -a work --all --limit 200 -o csv
```

### Options

```
      --filter string   OData filter ($filter), e.g. "isRead eq false"
      --folder string   mail folder: well-known name (inbox, sentitems, …) or id
  -h, --help            help for list
      --search string   KQL search query ($search); not combinable with --filter
      --select string   override the $select field list
      --top int         page size requested from Graph ($top, max 1000)
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

