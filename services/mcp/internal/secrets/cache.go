package secrets

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// CachedStringSource loads each Secret once per Lambda execution environment.
// Provider factories can share one credential without duplicating AWS calls.
type CachedStringSource struct {
	source StringSource

	mu     sync.Mutex
	values map[string]string
}

func NewCachedStringSource(source StringSource) *CachedStringSource {
	return &CachedStringSource{
		source: source,
		values: make(map[string]string),
	}
}

func (s *CachedStringSource) GetString(ctx context.Context, secretID string) (string, error) {
	if secretID == "" {
		return "", errors.New("secret ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if value, found := s.values[secretID]; found {
		return value, nil
	}
	if s.source == nil {
		return "", errors.New("secret source is not configured")
	}
	value, err := s.source.GetString(ctx, secretID)
	if err != nil {
		return "", fmt.Errorf("get secret %q: %w", secretID, err)
	}
	if value == "" {
		return "", errors.New("secret value is empty")
	}
	s.values[secretID] = value
	return value, nil
}
