package auth

import (
	"time"
)

// TokenStoreOption is the configuration options type for token store
type TokenStoreOption func(s *TokenStore)

// WithTokenStoreGCInterval returns option that sets token store garbage collection interval
func WithTokenStoreGCInterval(gcInterval time.Duration) TokenStoreOption {
	return func(s *TokenStore) {
		s.gcInterval = gcInterval
	}
}
