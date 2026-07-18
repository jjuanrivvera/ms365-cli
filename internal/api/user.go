package api

import (
	"context"
	"encoding/json"
	"net/url"
)

// meSelect trims /me to the identity fields a whoami actually needs.
const meSelect = "id,displayName,userPrincipalName,mail,jobTitle,officeLocation,mobilePhone,businessPhones"

// Me GETs the signed-in user's profile — the whoami/verification endpoint (User.Read).
func (c *Client) Me(ctx context.Context) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("$select", meSelect)
	status, _, body, err := c.Do(ctx, "GET", "me", q, nil, nil)
	if err != nil {
		return nil, err
	}
	if status == 0 { // dry-run
		return nil, nil
	}
	return body, nil
}
