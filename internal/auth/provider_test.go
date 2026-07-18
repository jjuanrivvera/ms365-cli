package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasScope(t *testing.T) {
	granted := []string{"User.Read", "Mail.Read", "Mail.Send"}

	assert.True(t, HasScope(granted, "Mail.Send"))
	assert.True(t, HasScope(granted, "mail.send"), "scope match must be case-insensitive")
	assert.False(t, HasScope(granted, "Calendars.ReadWrite"))

	// Entra sometimes returns resource-URI-qualified scopes.
	qualified := []string{"https://graph.microsoft.com/Mail.Send", "https://graph.microsoft.com/User.Read"}
	assert.True(t, HasScope(qualified, "Mail.Send"))
	assert.False(t, HasScope(qualified, "Mail.Read"))

	// No scope metadata ⇒ fail open (Graph enforces; the pre-check must not block blind).
	assert.True(t, HasScope(nil, "Mail.Send"))
	assert.True(t, HasScope([]string{}, "Mail.Send"))
}
