## ms365 mcp vscode

Manage VSCode MCP servers

### Synopsis

Manage MCP server configuration for Visual Studio Code

### Options

```
  -h, --help   help for vscode
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
* [ms365 mcp vscode disable](ms365_mcp_vscode_disable.md)	 - Remove server from VSCode config
* [ms365 mcp vscode enable](ms365_mcp_vscode_enable.md)	 - Add server to VSCode config
* [ms365 mcp vscode list](ms365_mcp_vscode_list.md)	 - Show VSCode MCP servers

