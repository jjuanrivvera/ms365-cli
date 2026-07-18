## ms365 mcp

MCP server management

### Synopsis

Manage MCP servers for AI assistants and code editors

### Options

```
  -h, --help   help for mcp
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
* [ms365 mcp claude](ms365_mcp_claude.md)	 - Manage Claude Desktop MCP servers
* [ms365 mcp cursor](ms365_mcp_cursor.md)	 - Manage Cursor MCP servers
* [ms365 mcp start](ms365_mcp_start.md)	 - Start the MCP server
* [ms365 mcp stream](ms365_mcp_stream.md)	 - Stream the MCP server over HTTP
* [ms365 mcp tools](ms365_mcp_tools.md)	 - Export tools as JSON
* [ms365 mcp vscode](ms365_mcp_vscode.md)	 - Manage VSCode MCP servers

