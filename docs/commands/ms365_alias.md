## ms365 alias

Manage user-defined command aliases

### Synopsis

Define shorthand commands. Aliases are expanded before parsing and can never shadow a built-in.

### Options

```
  -h, --help   help for alias
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
* [ms365 alias list](ms365_alias_list.md)	 - List aliases
* [ms365 alias remove](ms365_alias_remove.md)	 - Remove an alias
* [ms365 alias set](ms365_alias_set.md)	 - Create or update an alias

