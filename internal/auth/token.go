package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type tokenEntry struct {
	expiresAt time.Time
}

// TokenStore is a thread-safe in-memory store for bearer tokens.
type TokenStore struct {
	mu    sync.RWMutex
	store map[string]tokenEntry
}

func NewTokenStore() *TokenStore {
	return &TokenStore{store: make(map[string]tokenEntry)}
}

// Generate creates a new random token, stores it with the given TTL, and returns it.
func (s *TokenStore) Generate(ttl time.Duration) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	s.mu.Lock()
	s.store[token] = tokenEntry{expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()

	return token, nil
}

// IsValid reports whether the given token exists and has not expired.
func (s *TokenStore) IsValid(token string) bool {
	valid, _ := s.Validate(token)
	return valid
}

// Validate reports whether the token is valid and why validation failed.
func (s *TokenStore) Validate(token string) (bool, string) {
	if token == "" {
		return false, "empty_token"
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.store[token]
	if !ok {
		return false, "token_not_found"
	}
	if time.Now().After(entry.expiresAt) {
		return false, "token_expired"
	}
	return true, ""
}
