## ms365 mcp cursor list

Show Cursor MCP servers

### Synopsis

Show all MCP servers configured in Cursor

```
ms365 mcp cursor list [flags]
```

### Options

```
      --config-path string   Path to Cursor config file
  -h, --help                 help for list
      --workspace            List from workspace settings (.cursor/mcp.json) instead of user settings
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

* [ms365 mcp cursor](ms365_mcp_cursor.md)	 - Manage Cursor MCP servers

