package auth

import (
	"context"
	"errors"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

// cacheAccessor persists the MSAL token cache per profile through a Store (OS keyring with
// the encrypted-file fallback). Profiles never share a cache entry, so `-a personal` and
// `-a work` can hold two live sessions side by side.
type cacheAccessor struct {
	store   Store
	profile string
}

var _ cache.ExportReplace = (*cacheAccessor)(nil)

// Replace loads the serialized cache from storage into MSAL's in-memory cache. A missing
// entry is a fresh profile, not an error.
func (a *cacheAccessor) Replace(_ context.Context, u cache.Unmarshaler, _ cache.ReplaceHints) error {
	data, err := a.store.Get(ProfileKey(a.profile))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	return u.Unmarshal([]byte(data))
}

// Export writes MSAL's serialized cache back to storage after every token change.
func (a *cacheAccessor) Export(_ context.Context, m cache.Marshaler, _ cache.ExportHints) error {
	data, err := m.Marshal()
	if err != nil {
		return err
	}
	return a.store.Set(ProfileKey(a.profile), string(data))
}
