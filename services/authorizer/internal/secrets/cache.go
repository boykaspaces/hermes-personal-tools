package secrets

import (
	"context"
	"errors"
	"sync"
	"time"
)

type FetchFunc func(context.Context) (string, error)

type CachedString struct {
	fetch FetchFunc
	ttl   time.Duration

	mu        sync.Mutex
	value     string
	expiresAt time.Time
}

func NewCachedString(fetch FetchFunc, ttl time.Duration) *CachedString {
	return &CachedString{fetch: fetch, ttl: ttl}
}

func (c *CachedString) Get(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.value != "" && time.Now().Before(c.expiresAt) {
		return c.value, nil
	}
	value, err := c.fetch(ctx)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New("secret value is empty")
	}
	c.value = value
	c.expiresAt = time.Now().Add(c.ttl)
	return value, nil
}
