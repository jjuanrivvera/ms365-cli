package auth

import (
	"testing"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// fakeMarshaler stands in for MSAL's serialized cache.
type fakeMarshaler struct{ data []byte }

func (f *fakeMarshaler) Marshal() ([]byte, error) { return f.data, nil }

type fakeUnmarshaler struct{ got []byte }

func (f *fakeUnmarshaler) Unmarshal(b []byte) error { f.got = append([]byte{}, b...); return nil }

func TestCacheAccessor_RoundTrip(t *testing.T) {
	keyring.MockInit()
	s := New(t.TempDir())
	a := &cacheAccessor{store: s, profile: "personal"}

	require.NoError(t, a.Export(t.Context(), &fakeMarshaler{data: []byte(`{"msal":"cache"}`)}, cache.ExportHints{}))

	u := &fakeUnmarshaler{}
	require.NoError(t, a.Replace(t.Context(), u, cache.ReplaceHints{}))
	assert.Equal(t, `{"msal":"cache"}`, string(u.got))

	// The entry is namespaced per profile — a different profile sees nothing.
	other := &cacheAccessor{store: s, profile: "work"}
	u2 := &fakeUnmarshaler{}
	require.NoError(t, other.Replace(t.Context(), u2, cache.ReplaceHints{}))
	assert.Nil(t, u2.got, "profiles must never share a token cache")
}

func TestCacheAccessor_MissingEntryIsFreshProfile(t *testing.T) {
	keyring.MockInit()
	a := &cacheAccessor{store: New(t.TempDir()), profile: "nobody"}
	u := &fakeUnmarshaler{}
	require.NoError(t, a.Replace(t.Context(), u, cache.ReplaceHints{}))
	assert.Nil(t, u.got)
}

func TestProfileKey(t *testing.T) {
	assert.Equal(t, "profile-personal", ProfileKey("personal"))
}
