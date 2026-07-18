// Package commands wires the cobra command tree. root.go owns the global flags, the shared
// Graph client factory, and the single render() path used by every resource command. The
// tree is built fresh per NewRootCmd() call so tests never leak flag state across cases.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/ms365-cli/internal/api"
	"github.com/jjuanrivvera/ms365-cli/internal/auth"
	"github.com/jjuanrivvera/ms365-cli/internal/config"
	"github.com/jjuanrivvera/ms365-cli/internal/output"
)

// globalFlags holds the persistent flag values for one command tree.
type globalFlags struct {
	outputFormat string
	account      string // the profile selector (-a/--account; hidden --profile alias)
	baseURL      string
	clientID     string
	timezone     string
	dryRun       bool
	showToken    bool
	verbose      bool
	noColor      bool
	columns      []string
	quiet        bool
	jq           string

	// list flags (read by every list command)
	all   bool
	limit int
}

// deps carries the per-tree state into every command builder.
type deps struct {
	gf *globalFlags

	// overridable in tests
	loadConfig func() (*config.Config, error)
	store      func() auth.Store
	// provider builds the auth provider for a profile; tests inject fakes.
	provider func(clientID, authority, profile string, scopes []string) auth.Provider
	// out overrides where dry-run curls go (tests capture it; default os.Stdout).
	out io.Writer
}

func newDeps() *deps {
	return &deps{
		gf:         &globalFlags{},
		loadConfig: config.Load,
		store: func() auth.Store {
			dir, err := config.Dir()
			if err != nil {
				dir = "."
			}
			return auth.New(dir)
		},
	}
}

// authProvider resolves the Provider for a profile, honoring a test override.
func (d *deps) authProvider(clientID, profile string) auth.Provider {
	if d.provider != nil {
		return d.provider(clientID, auth.DefaultAuthority, profile, auth.DefaultScopes)
	}
	return auth.NewMSALProvider(clientID, auth.DefaultAuthority, profile, auth.DefaultScopes, d.store())
}

// NewRootCmd assembles the full command tree. main.go calls
// NewRootCmd().ExecuteContext(ctx) with a signal.NotifyContext so Ctrl-C cancels
// in-flight work (including the device-code wait).
func NewRootCmd() *cobra.Command { return newRootCmd(newDeps()) }

// registrars build the resource commands; resource files append from init().
var registrars []func(d *deps) *cobra.Command

// metaRegistrars register the non-resource commands (auth, config, doctor, …).
var metaRegistrars []func(d *deps) *cobra.Command

// newRootCmd is the deps-injected assembly used by tests (fake provider, temp config).
func newRootCmd(d *deps) *cobra.Command {
	root := &cobra.Command{
		Use:   "ms365",
		Short: "A fast, scriptable CLI for Microsoft 365 (Microsoft Graph)",
		Long: `ms365 drives Microsoft 365 from the terminal via the Microsoft Graph API: read and
send mail, manage your calendar, and inspect your profile — with named accounts so a
personal Outlook.com sign-in and a work/school tenant live side by side and never cross
tokens.

Sign-in is the OAuth device-code flow (no secrets stored beyond MSAL's refresh tokens,
which live in your OS keyring). The default sign-in is read-only; write commands tell you
the extra scope to grant (e.g. ` + "`auth login --scopes Mail.Send`" + `).

Examples:
  ms365 auth login -a personal
  ms365 mail list -a personal --top 20
  ms365 mail list --folder inbox --search "invoice"
  ms365 mail get <message-id>
  ms365 mail send --to ana@example.com --subject "Lunch?" --body "12:30?"
  ms365 calendar events --from 2026-07-20 --to 2026-07-27
  ms365 calendar create --subject "1:1" --from 2026-07-21T10:00 --to 2026-07-21T10:30
  ms365 me -o json`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if d.gf.outputFormat != "" && !output.Format(d.gf.outputFormat).Valid() {
				return fmt.Errorf("unknown output format %q (want table|json|yaml|csv|id)", d.gf.outputFormat)
			}
			if d.gf.account != "" {
				if err := config.ValidateProfileName(d.gf.account); err != nil {
					return err
				}
			}
			return nil
		},
	}
	registerGlobalFlags(root, d.gf)

	for _, build := range registrars {
		root.AddCommand(build(d))
	}
	for _, build := range metaRegistrars {
		root.AddCommand(build(d))
	}
	return root
}

func registerGlobalFlags(root *cobra.Command, gf *globalFlags) {
	pf := root.PersistentFlags()
	pf.StringVarP(&gf.outputFormat, "output", "o", "", "output format: table|json|yaml|csv|id")
	// --account is the profile selector (a profile IS a Microsoft account here); --profile
	// stays as a hidden alias so fleet muscle memory and scripts keep working.
	pf.StringVarP(&gf.account, "account", "a", "", "named account (profile) to use")
	pf.StringVar(&gf.account, "profile", "", "alias for --account")
	_ = pf.MarkHidden("profile")
	pf.StringVar(&gf.baseURL, "base-url", "", "Microsoft Graph base URL override (sovereign clouds)")
	pf.StringVar(&gf.clientID, "client-id", "", "Entra app (client) ID override — defaults to the embedded Microsoft Graph Command Line Tools app")
	pf.StringVar(&gf.timezone, "timezone", "", `return mail/calendar datetimes in this timezone (Prefer: outlook.timezone, e.g. "America/Caracas")`)
	pf.BoolVar(&gf.dryRun, "dry-run", false, "print the equivalent curl and make no request")
	pf.BoolVar(&gf.showToken, "show-token", false, "reveal the bearer token in dry-run output")
	pf.BoolVarP(&gf.verbose, "verbose", "v", false, "verbose request logging (stderr)")
	pf.BoolVar(&gf.noColor, "no-color", false, "disable colored output")
	pf.StringSliceVar(&gf.columns, "columns", nil, "comma-separated columns to show")
	pf.BoolVar(&gf.quiet, "quiet", false, "suppress non-essential chatter")
	pf.StringVar(&gf.jq, "jq", "", "gojq expression applied to the response before rendering")

	pf.BoolVar(&gf.all, "all", false, "follow @odata.nextLink until exhausted (list commands)")
	pf.IntVar(&gf.limit, "limit", 0, "max items to return across pages (list commands)")
}

// resolveProfile returns the active profile name and config.
func (d *deps) resolveProfile() (string, *config.Config, error) {
	cfg, err := d.loadConfig()
	if err != nil {
		return "", nil, err
	}
	return cfg.ResolveProfileName(d.gf.account), cfg, nil
}

// resolveClientID applies flag > env > profile config > embedded default.
func (d *deps) resolveClientID(cfg *config.Config, profileName string) string {
	prof, _ := cfg.Profile(profileName)
	return config.FirstNonEmpty(d.gf.clientID, os.Getenv("MS365_CLIENT_ID"), prof.ClientID, auth.DefaultClientID)
}

// getAPIClient builds an authenticated Graph client for the ACTIVE profile, honoring
// flag > env > config precedence for every knob.
func (d *deps) getAPIClient() (*api.Client, *config.Config, error) {
	return d.getAPIClientScoped("")
}

// getAPIClientScoped is getAPIClient plus a delegated-scope pre-check for write commands:
// DefaultScopes stay read-only by design (DECISIONS.md #5/#18), so a token minted by a
// plain `auth login` lacks Mail.Send / Calendars.ReadWrite. Checking the cached grant here
// turns Graph's opaque 403 into the exact re-login command the user needs. The check runs
// inside the TokenFunc — which --dry-run never invokes — so dry-run keeps working unsigned.
func (d *deps) getAPIClientScoped(scope string) (*api.Client, *config.Config, error) {
	profileName, cfg, err := d.resolveProfile()
	if err != nil {
		return nil, nil, err
	}
	prof, _ := cfg.Profile(profileName)

	baseURL := config.FirstNonEmpty(d.gf.baseURL, os.Getenv("MS365_BASE_URL"), prof.BaseURL, api.DefaultBaseURL)
	if err := config.ValidateBaseURL(baseURL); err != nil {
		return nil, nil, err
	}
	tz := config.FirstNonEmpty(d.gf.timezone, os.Getenv("MS365_TIMEZONE"), prof.Timezone)

	provider := d.authProvider(d.resolveClientID(cfg, profileName), profileName)
	c := api.New(baseURL, func(ctx context.Context) (string, error) {
		tok, err := provider.TokenSilent(ctx)
		if err != nil {
			return "", fmt.Errorf("no valid token for account %q — run `ms365 auth login -a %s` (%w)", profileName, profileName, err)
		}
		if scope != "" && !auth.HasScope(tok.Scopes, scope) {
			return "", fmt.Errorf("account %q is signed in without the %s scope — run: ms365 auth login -a %s --scopes %s", profileName, scope, profileName, scope)
		}
		return tok.AccessToken, nil
	},
		api.WithDryRun(d.gf.dryRun, d.stdout()),
		api.WithTimezone(tz),
	)
	c.ShowToken = d.gf.showToken
	c.Verbose = d.gf.verbose
	c.VerboseOut = os.Stderr
	return c, cfg, nil
}

func (d *deps) stdout() io.Writer {
	if d.out != nil {
		return d.out
	}
	return os.Stdout
}

// render is the single output path for every command: normalize v to JSON, then hand it to
// the shared renderer with the resolved global flags.
func (d *deps) render(cmd *cobra.Command, v any, defaultColumns []string) error {
	raw, ok := v.(json.RawMessage)
	if !ok {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		raw = b
	}
	format := output.Format(config.FirstNonEmpty(d.gf.outputFormat, string(output.FormatTable)))
	cols := normalizeColumns(d.gf.columns)
	if len(cols) == 0 && format != output.FormatID {
		// Default columns shape tables/CSV; `-o id` must keep its id-picking heuristic
		// (the default column list rarely leads with id).
		cols = defaultColumns
	}
	return output.Render(raw, output.Options{
		Format:  format,
		Columns: cols,
		NoColor: d.gf.noColor,
		Quiet:   d.gf.quiet,
		JQ:      d.gf.jq,
		Out:     cmd.OutOrStdout(),
		Err:     cmd.ErrOrStderr(),
	})
}

func normalizeColumns(cols []string) []string {
	var out []string
	for _, c := range cols {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}
