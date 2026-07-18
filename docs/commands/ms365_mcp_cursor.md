## ms365 mcp cursor

Manage Cursor MCP servers

### Synopsis

Manage MCP server configuration for Cursor

### Options

```
  -h, --help   help for cursor
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

* [ms365 mcp](ms365_mcp.md)	 - MCP server management
* [ms365 mcp cursor disable](ms365_mcp_cursor_disable.md)	 - Remove server from Cursor config
* [ms365 mcp cursor enable](ms365_mcp_cursor_enable.md)	 - Add server to Cursor config
* [ms365 mcp cursor list](ms365_mcp_cursor_list.md)	 - Show Cursor MCP servers

