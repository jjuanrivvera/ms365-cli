package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// calendarDefaultSelect keeps event rows tabular; the full JSON is one -o json away.
const calendarDefaultSelect = "id,subject,start,end,location,organizer,isAllDay,isCancelled,onlineMeetingUrl"

// CalendarViewOptions parameterize the /me/calendarView time-window query.
type CalendarViewOptions struct {
	From   time.Time
	To     time.Time
	Top    int
	Select string
	Limit  int
	All    bool
}

// CalendarView GETs /me/calendarView between From and To. Graph requires both bounds as
// ISO 8601 and expands recurring events into occurrences — which is why this endpoint,
// not /me/events, answers "what is on my calendar this week".
func (c *Client) CalendarView(ctx context.Context, opts CalendarViewOptions) ([]json.RawMessage, error) {
	if !opts.From.Before(opts.To) {
		return nil, fmt.Errorf("--from (%s) must be before --to (%s)", opts.From.Format(time.RFC3339), opts.To.Format(time.RFC3339))
	}
	q := url.Values{}
	q.Set("startDateTime", opts.From.UTC().Format(time.RFC3339))
	q.Set("endDateTime", opts.To.UTC().Format(time.RFC3339))
	sel := opts.Select
	if sel == "" {
		sel = calendarDefaultSelect
	}
	q.Set("$select", sel)
	q.Set("$orderby", "start/dateTime")
	if opts.Top > 0 {
		q.Set("$top", strconv.Itoa(opts.Top))
	}
	return c.List(ctx, "me/calendarView", q, ListOptions{Limit: opts.Limit, All: opts.All})
}

// EventListOptions parameterize the /me/events listing (event masters, not occurrences).
type EventListOptions struct {
	Top    int
	Filter string
	Select string
	Limit  int
	All    bool
}

// EventsList GETs /me/events — the event masters (single + recurrence series), unlike
// CalendarView which expands occurrences.
func (c *Client) EventsList(ctx context.Context, opts EventListOptions) ([]json.RawMessage, error) {
	q := url.Values{}
	sel := opts.Select
	if sel == "" {
		sel = calendarDefaultSelect
	}
	q.Set("$select", sel)
	if opts.Top > 0 {
		q.Set("$top", strconv.Itoa(opts.Top))
	}
	if opts.Filter != "" {
		q.Set("$filter", opts.Filter)
	}
	return c.List(ctx, "me/events", q, ListOptions{Limit: opts.Limit, All: opts.All})
}
