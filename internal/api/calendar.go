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

// EventDateTime is Graph's dateTimeTimeZone: a wall-clock time paired with an IANA/Windows
// timezone name (NOT an offset — Exchange resolves DST from the zone).
type EventDateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

// EventFields are the writable event properties the CLI exposes. Pointer fields distinguish
// "not set" from a zero value so PATCH sends only the flags the user actually passed.
type EventFields struct {
	Subject       *string
	Start         *EventDateTime
	End           *EventDateTime
	Location      *string
	Body          *string
	Attendees     []string // nil = untouched; PATCH replaces the whole attendee list
	OnlineMeeting *bool
}

// payload converts the set fields into Graph's event resource shape.
func (f EventFields) payload() map[string]any {
	p := map[string]any{}
	if f.Subject != nil {
		p["subject"] = *f.Subject
	}
	if f.Start != nil {
		p["start"] = *f.Start
	}
	if f.End != nil {
		p["end"] = *f.End
	}
	if f.Location != nil {
		p["location"] = map[string]string{"displayName": *f.Location}
	}
	if f.Body != nil {
		p["body"] = map[string]string{"contentType": "text", "content": *f.Body}
	}
	if f.Attendees != nil {
		att := make([]map[string]any, 0, len(f.Attendees))
		for _, a := range f.Attendees {
			att = append(att, map[string]any{
				"emailAddress": map[string]string{"address": a},
				"type":         "required",
			})
		}
		p["attendees"] = att
	}
	if f.OnlineMeeting != nil {
		p["isOnlineMeeting"] = *f.OnlineMeeting
		if *f.OnlineMeeting {
			// isOnlineMeeting alone is not enough — Graph needs the provider to
			// actually provision a Teams link.
			p["onlineMeetingProvider"] = "teamsForBusiness"
		}
	}
	return p
}

// EventCreate POSTs /me/events and returns the created event resource.
func (c *Client) EventCreate(ctx context.Context, fields EventFields) (json.RawMessage, error) {
	if fields.Subject == nil || fields.Start == nil || fields.End == nil {
		return nil, fmt.Errorf("--subject, --from, and --to are required to create an event")
	}
	body, err := json.Marshal(fields.payload())
	if err != nil {
		return nil, err
	}
	status, _, resp, err := c.Do(ctx, "POST", "me/events", nil, body, nil)
	if err != nil {
		return nil, err
	}
	if status == 0 { // dry-run
		return nil, nil
	}
	return resp, nil
}

// EventUpdate PATCHes /me/events/{id} with ONLY the set fields and returns the updated
// event resource.
func (c *Client) EventUpdate(ctx context.Context, id string, fields EventFields) (json.RawMessage, error) {
	if err := validateIDSegment(id); err != nil {
		return nil, fmt.Errorf("invalid event id: %w", err)
	}
	payload := fields.payload()
	if len(payload) == 0 {
		return nil, fmt.Errorf("nothing to update — pass at least one field flag (--subject, --from, --to, --location, --body, --attendee, --online-meeting)")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	status, _, resp, err := c.Do(ctx, "PATCH", "me/events/"+url.PathEscape(id), nil, body, nil)
	if err != nil {
		return nil, err
	}
	if status == 0 { // dry-run
		return nil, nil
	}
	return resp, nil
}

// EventDelete DELETEs /me/events/{id}. Graph returns 204 with no body.
func (c *Client) EventDelete(ctx context.Context, id string) error {
	if err := validateIDSegment(id); err != nil {
		return fmt.Errorf("invalid event id: %w", err)
	}
	_, _, _, err := c.Do(ctx, "DELETE", "me/events/"+url.PathEscape(id), nil, nil, nil)
	return err
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
