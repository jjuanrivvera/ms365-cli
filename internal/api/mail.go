package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Recipient is Graph's emailAddress wrapper.
type Recipient struct {
	EmailAddress struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"emailAddress"`
}

// ItemBody is a mail/event body with its content type.
type ItemBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

// Message is the subset of Graph's message resource the CLI renders directly; the full
// JSON is always available via -o json.
type Message struct {
	ID               string      `json:"id"`
	Subject          string      `json:"subject"`
	From             Recipient   `json:"from"`
	ToRecipients     []Recipient `json:"toRecipients"`
	ReceivedDateTime string      `json:"receivedDateTime"`
	BodyPreview      string      `json:"bodyPreview"`
	Body             ItemBody    `json:"body"`
	IsRead           bool        `json:"isRead"`
	HasAttachments   bool        `json:"hasAttachments"`
	WebLink          string      `json:"webLink"`
}

// MailListOptions map the mail list flags onto Graph OData query parameters.
type MailListOptions struct {
	Folder string // well-known name (inbox, sentitems, drafts, …) or a folder id
	Top    int    // $top page size
	Search string // $search (KQL; server rejects combining with $filter/$orderby)
	Filter string // $filter
	Select string // $select override
	Limit  int
	All    bool
}

// mailDefaultSelect keeps list payloads light; `mail get` fetches the full message.
const mailDefaultSelect = "id,subject,from,receivedDateTime,isRead,hasAttachments,bodyPreview"

// MailList GETs /me/messages (or /me/mailFolders/{folder}/messages).
func (c *Client) MailList(ctx context.Context, opts MailListOptions) ([]json.RawMessage, error) {
	if opts.Search != "" && opts.Filter != "" {
		return nil, fmt.Errorf("--search and --filter cannot be combined (Microsoft Graph rejects $search with $filter on messages)")
	}
	path := "me/messages"
	if opts.Folder != "" {
		if err := validateIDSegment(opts.Folder); err != nil {
			return nil, fmt.Errorf("invalid folder: %w", err)
		}
		path = "me/mailFolders/" + url.PathEscape(opts.Folder) + "/messages"
	}
	q := url.Values{}
	sel := opts.Select
	if sel == "" {
		sel = mailDefaultSelect
	}
	q.Set("$select", sel)
	if opts.Top > 0 {
		q.Set("$top", strconv.Itoa(opts.Top))
	}
	switch {
	case opts.Search != "":
		// Graph requires the KQL clause quoted inside the parameter value.
		q.Set("$search", `"`+strings.ReplaceAll(opts.Search, `"`, `\"`)+`"`)
	case opts.Filter != "":
		q.Set("$filter", opts.Filter)
	}
	return c.List(ctx, path, q, ListOptions{Limit: opts.Limit, All: opts.All})
}

// MailGet GETs one message. When asText is true it asks Exchange to down-convert the body
// to plain text via `Prefer: outlook.body-content-type="text"`.
func (c *Client) MailGet(ctx context.Context, id string, asText bool) (*Message, json.RawMessage, error) {
	if err := validateIDSegment(id); err != nil {
		return nil, nil, fmt.Errorf("invalid message id: %w", err)
	}
	extra := map[string]string{}
	if asText {
		extra["Prefer"] = `outlook.body-content-type="text"`
	}
	status, _, body, err := c.Do(ctx, "GET", "me/messages/"+url.PathEscape(id), nil, nil, extra)
	if err != nil {
		return nil, nil, err
	}
	if status == 0 { // dry-run
		return nil, nil, nil
	}
	var m Message
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, nil, fmt.Errorf("decode message: %w", err)
	}
	return &m, body, nil
}

// validateIDSegment rejects values that would break out of the URL path segment (Graph ids
// are base64url-ish and never contain slashes).
func validateIDSegment(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("empty")
	}
	if strings.ContainsAny(id, "/\\?#") {
		return fmt.Errorf("%q contains a path metacharacter", id)
	}
	return nil
}
