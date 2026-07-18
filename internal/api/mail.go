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

// MailSendOptions describe one outgoing message for /me/sendMail.
type MailSendOptions struct {
	To, Cc, Bcc []string
	Subject     string
	Body        string
	HTML        bool // body contentType: html instead of text
	SaveToSent  bool // Graph's saveToSentItems
}

// MailSend POSTs /me/sendMail (fire-and-forget: Graph queues the message and returns 202
// with no body, so there is nothing to render on success).
func (c *Client) MailSend(ctx context.Context, opts MailSendOptions) error {
	if len(opts.To) == 0 {
		return fmt.Errorf("at least one --to recipient is required")
	}
	for _, lst := range [][]string{opts.To, opts.Cc, opts.Bcc} {
		for _, a := range lst {
			if !strings.Contains(a, "@") {
				return fmt.Errorf("invalid recipient %q (want an email address)", a)
			}
		}
	}
	contentType := "text"
	if opts.HTML {
		contentType = "html"
	}
	msg := map[string]any{
		"subject":      opts.Subject,
		"body":         map[string]string{"contentType": contentType, "content": opts.Body},
		"toRecipients": recipients(opts.To),
	}
	if len(opts.Cc) > 0 {
		msg["ccRecipients"] = recipients(opts.Cc)
	}
	if len(opts.Bcc) > 0 {
		msg["bccRecipients"] = recipients(opts.Bcc)
	}
	body, err := json.Marshal(map[string]any{"message": msg, "saveToSentItems": opts.SaveToSent})
	if err != nil {
		return err
	}
	_, _, _, err = c.Do(ctx, "POST", "me/sendMail", nil, body, nil)
	return err
}

// MailReply POSTs /me/messages/{id}/reply (or /replyAll when all is true). Exchange builds
// the quoted thread server-side; comment is prepended as the reply text.
func (c *Client) MailReply(ctx context.Context, id, comment string, all bool) error {
	if err := validateIDSegment(id); err != nil {
		return fmt.Errorf("invalid message id: %w", err)
	}
	action := "reply"
	if all {
		action = "replyAll"
	}
	body, err := json.Marshal(map[string]string{"comment": comment})
	if err != nil {
		return err
	}
	_, _, _, err = c.Do(ctx, "POST", "me/messages/"+url.PathEscape(id)+"/"+action, nil, body, nil)
	return err
}

// recipients wraps plain addresses in Graph's recipient envelope.
func recipients(addrs []string) []map[string]any {
	out := make([]map[string]any, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, map[string]any{"emailAddress": map[string]string{"address": strings.TrimSpace(a)}})
	}
	return out
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
