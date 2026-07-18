package commands

import (
	"fmt"
	"strings"
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
		calCmd.AddCommand(newCalendarEventsCmd(d), newCalendarListCmd(d),
			newCalendarCreateCmd(d), newCalendarUpdateCmd(d), newCalendarDeleteCmd(d))
		return calCmd
	})
}

// scopeCalendarWrite is the delegated scope the calendar write verbs need beyond
// DefaultScopes.
const scopeCalendarWrite = "Calendars.ReadWrite"

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

// eventFlags are the writable-event flags shared by `calendar create` and
// `calendar update` — one definition so the two surfaces can never drift.
type eventFlags struct {
	subject, from, to, location, body string
	attendees                         []string
	onlineMeeting                     bool
}

func registerEventFlags(cmd *cobra.Command, f *eventFlags) {
	cmd.Flags().StringVar(&f.subject, "subject", "", "event subject")
	cmd.Flags().StringVar(&f.from, "from", "", "start (YYYY-MM-DDTHH:MM, wall-clock in --timezone)")
	cmd.Flags().StringVar(&f.to, "to", "", "end (YYYY-MM-DDTHH:MM, wall-clock in --timezone)")
	cmd.Flags().StringVar(&f.location, "location", "", "location display name")
	cmd.Flags().StringVar(&f.body, "body", "", "event body text")
	cmd.Flags().StringSliceVar(&f.attendees, "attendee", nil, "attendee email (repeatable or comma-separated)")
	cmd.Flags().BoolVar(&f.onlineMeeting, "online-meeting", false, "provision a Teams online meeting")
}

// fields maps ONLY the flags the user actually passed (cobra Changed) onto EventFields, so
// `calendar update` PATCHes just those properties. tz names the wall-clock zone for
// --from/--to (empty means UTC).
func (f *eventFlags) fields(cmd *cobra.Command, tz string) (api.EventFields, error) {
	var out api.EventFields
	fl := cmd.Flags()
	if fl.Changed("subject") {
		out.Subject = &f.subject
	}
	if fl.Changed("from") {
		start, err := parseEventDateTime(f.from, tz)
		if err != nil {
			return out, fmt.Errorf("--from: %w", err)
		}
		out.Start = start
	}
	if fl.Changed("to") {
		end, err := parseEventDateTime(f.to, tz)
		if err != nil {
			return out, fmt.Errorf("--to: %w", err)
		}
		out.End = end
	}
	if fl.Changed("location") {
		out.Location = &f.location
	}
	if fl.Changed("body") {
		out.Body = &f.body
	}
	if fl.Changed("attendee") {
		out.Attendees = f.attendees
	}
	if fl.Changed("online-meeting") {
		out.OnlineMeeting = &f.onlineMeeting
	}
	return out, nil
}

// parseEventDateTime reads a WALL-CLOCK datetime — Graph's dateTimeTimeZone pairs the
// literal clock time with a zone NAME and lets Exchange resolve DST, so no offset math
// happens here.
func parseEventDateTime(s, tz string) (*api.EventDateTime, error) {
	if tz == "" {
		tz = "UTC"
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &api.EventDateTime{DateTime: t.Format("2006-01-02T15:04:05"), TimeZone: tz}, nil
		}
	}
	return nil, fmt.Errorf("invalid datetime %q (want YYYY-MM-DDTHH:MM or YYYY-MM-DD)", s)
}

func newCalendarCreateCmd(d *deps) *cobra.Command {
	var f eventFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a calendar event (Calendars.ReadWrite)",
		Long: `Create an event in the default calendar via POST /me/events. --from/--to are
wall-clock times interpreted in --timezone (default UTC); attendees receive an
invitation. --online-meeting provisions a Teams meeting link.

Creating events needs the Calendars.ReadWrite delegated scope, which the default
sign-in does NOT request — grant it once per account with:
  ms365 auth login -a <account> --scopes Calendars.ReadWrite`,
		Example: `  ms365 calendar create --subject "1:1 Ana" --from 2026-07-21T10:00 --to 2026-07-21T10:30
  ms365 calendar create --subject "Sprint review" --from 2026-07-24T15:00 --to 2026-07-24T16:00 \
    --timezone "America/Caracas" --attendee ana@x.com --attendee bo@x.com --online-meeting
  ms365 calendar create --subject "Focus block" --from 2026-07-22T09:00 --to 2026-07-22T11:00 --location "Home office"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fields, err := f.fields(cmd, d.gf.timezone)
			if err != nil {
				return err
			}
			c, _, err := d.getAPIClientScoped(scopeCalendarWrite)
			if err != nil {
				return err
			}
			created, err := c.EventCreate(cmd.Context(), fields)
			if err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			return d.render(cmd, created, calendarColumns)
		},
	}
	registerEventFlags(cmd, &f)
	return annotate(cmd, kindWrite)
}

func newCalendarUpdateCmd(d *deps) *cobra.Command {
	var f eventFlags
	cmd := &cobra.Command{
		Use:   "update <event-id>",
		Short: "Update a calendar event (Calendars.ReadWrite)",
		Long: `PATCH /me/events/{id} with ONLY the flags you pass — unset flags leave the event
untouched. --attendee REPLACES the whole attendee list. Get event ids from
'calendar list -o id' or 'calendar events -o id'.

Updating events needs the Calendars.ReadWrite delegated scope — grant it once per
account with:
  ms365 auth login -a <account> --scopes Calendars.ReadWrite`,
		Example: `  ms365 calendar update AAMkAGI2… --subject "1:1 Ana (moved)"
  ms365 calendar update AAMkAGI2… --from 2026-07-21T11:00 --to 2026-07-21T11:30 --timezone "America/Caracas"
  ms365 calendar update AAMkAGI2… --location "Room 3" --online-meeting=false`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fields, err := f.fields(cmd, d.gf.timezone)
			if err != nil {
				return err
			}
			c, _, err := d.getAPIClientScoped(scopeCalendarWrite)
			if err != nil {
				return err
			}
			updated, err := c.EventUpdate(cmd.Context(), args[0], fields)
			if err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			return d.render(cmd, updated, calendarColumns)
		},
	}
	registerEventFlags(cmd, &f)
	return annotate(cmd, kindWrite)
}

func newCalendarDeleteCmd(d *deps) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <event-id>",
		Short: "Delete a calendar event (Calendars.ReadWrite)",
		Long: `DELETE /me/events/{id}. Attendees receive a cancellation when you organize the
event. Irreversible — the command asks for confirmation unless --yes is passed.

Deleting events needs the Calendars.ReadWrite delegated scope — grant it once per
account with:
  ms365 auth login -a <account> --scopes Calendars.ReadWrite`,
		Example: `  ms365 calendar delete AAMkAGI2… --yes
  ms365 calendar list --filter "subject eq 'Old sync'" -o id | xargs -I{} ms365 calendar delete {} --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// --dry-run sends nothing, so there is nothing to confirm.
			if !yes && !d.gf.dryRun {
				ans, err := promptLine(cmd, fmt.Sprintf("Delete event %s? [y/N] ", args[0]))
				if err != nil {
					return err
				}
				if a := strings.ToLower(ans); a != "y" && a != "yes" {
					return fmt.Errorf("aborted — nothing deleted (pass --yes to skip confirmation)")
				}
			}
			c, _, err := d.getAPIClientScoped(scopeCalendarWrite)
			if err != nil {
				return err
			}
			if err := c.EventDelete(cmd.Context(), args[0]); err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted event %s.\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return annotate(cmd, kindDestructive)
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
