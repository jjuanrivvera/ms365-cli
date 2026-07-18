## ms365 config

Inspect and edit ms365 configuration

### Synopsis

The config file holds only non-secret settings (account profiles, aliases,
overrides). Tokens live in the OS keyring — never here.

### Options

```
  -h, --help   help for config
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
* [ms365 config list-profiles](ms365_config_list-profiles.md)	 - List configured accounts
* [ms365 config path](ms365_config_path.md)	 - Print the config file path
* [ms365 config set](ms365_config_set.md)	 - Set a per-account option (client_id, base_url, timezone)
* [ms365 config use](ms365_config_use.md)	 - Set the default account for future invocations
* [ms365 config view](ms365_config_view.md)	 - Show the resolved configuration

