package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/ms365-cli/internal/api"
)

// calendarColumns are the default table columns for event listings.
var calendarColumns = []string{"subject", "start.dateTime", "end.dateTime", "location.displayName", "organizer.emailAddress.address", "id"}

func init() {
	registrars = append(registrars, func(d *deps) *cobra.Command {
		calCmd := &cobra.Command{
			Use:     "calendar",
			Aliases: []string{"cal"},
			Short:   "Read your Outlook calendar (Calendars.Read)",
		}
		calCmd.AddCommand(newCalendarEventsCmd(d), newCalendarListCmd(d))
		return calCmd
	})
}

func newCalendarEventsCmd(d *deps) *cobra.Command {
	var from, to string
	var top int
	var sel string
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Show calendar events in a time window (recurring events expanded)",
		Long: `Query /me/calendarView between --from and --to (default: the next 7 days).
Unlike 'calendar list', this expands recurring events into their occurrences — it
answers "what is actually on my calendar".

Dates accept YYYY-MM-DD or full RFC 3339 timestamps. Combine with --timezone to get
start/end in your zone instead of UTC.`,
		Example: `  ms365 calendar events
  ms365 calendar events --from 2026-07-20 --to 2026-07-27
  ms365 calendar events --timezone "America/Caracas" -o json
  ms365 calendar events -a work --limit 50`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			now := time.Now()
			fromT, err := parseDateArg(from, now)
			if err != nil {
				return fmt.Errorf("--from: %w", err)
			}
			toT, err := parseDateArg(to, now.AddDate(0, 0, 7))
			if err != nil {
				return fmt.Errorf("--to: %w", err)
			}
			c, _, err := d.getAPIClient()
			if err != nil {
				return err
			}
			items, err := c.CalendarView(cmd.Context(), api.CalendarViewOptions{
				From: fromT, To: toT, Top: top, Select: sel,
				Limit: d.gf.limit, All: d.gf.all,
			})
			if err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			return d.render(cmd, rawItems(items), calendarColumns)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "window start (YYYY-MM-DD or RFC 3339; default now)")
	cmd.Flags().StringVar(&to, "to", "", "window end (YYYY-MM-DD or RFC 3339; default now+7d)")
	cmd.Flags().IntVar(&top, "top", 0, "page size requested from Graph ($top)")
	cmd.Flags().StringVar(&sel, "select", "", "override the $select field list")
	return annotate(cmd, kindRead)
}

func newCalendarListCmd(d *deps) *cobra.Command {
	var opts api.EventListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List event masters (/me/events)",
		Long: `List the events in the default calendar — the event MASTERS: single events and
recurrence series, without occurrence expansion. Use 'calendar events' for the
expanded time-window view.`,
		Example: `  ms365 calendar list --top 25
  ms365 calendar list --filter "isOrganizer eq true" -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, _, err := d.getAPIClient()
			if err != nil {
				return err
			}
			opts.Limit = d.gf.limit
			opts.All = d.gf.all
			items, err := c.EventsList(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			return d.render(cmd, rawItems(items), calendarColumns)
		},
	}
	cmd.Flags().IntVar(&opts.Top, "top", 0, "page size requested from Graph ($top)")
	cmd.Flags().StringVar(&opts.Filter, "filter", "", "OData filter ($filter)")
	cmd.Flags().StringVar(&opts.Select, "select", "", "override the $select field list")
	return annotate(cmd, kindRead)
}

// parseDateArg accepts YYYY-MM-DD (local midnight) or RFC 3339; empty falls back to def.
func parseDateArg(s string, def time.Time) (time.Time, error) {
	if s == "" {
		return def, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid date %q (want YYYY-MM-DD or RFC 3339)", s)
}
