package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type CSRFStore struct {
	mu     sync.RWMutex
	tokens map[string]csrfEntry
}

type csrfEntry struct {
	Token     string
	SessionID string
	ExpiresAt time.Time
}

func NewCSRFStore() *CSRFStore {
	store := &CSRFStore{
		tokens: make(map[string]csrfEntry),
	}
	go store.cleanupLoop()
	return store
}

func (s *CSRFStore) Generate(sessionID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	s.tokens[sessionID] = csrfEntry{
		Token:     token,
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	return token
}

func (s *CSRFStore) Validate(sessionID, token string) bool {
	s.mu.RLock()
	entry, ok := s.tokens[sessionID]
	s.mu.RUnlock()

	if !ok {
		return false
	}
	if time.Now().After(entry.ExpiresAt) {
		s.mu.Lock()
		delete(s.tokens, sessionID)
		s.mu.Unlock()
		return false
	}
	return entry.Token == token
}

func (s *CSRFStore) Remove(sessionID string) {
	s.mu.Lock()
	delete(s.tokens, sessionID)
	s.mu.Unlock()
}

func (s *CSRFStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, entry := range s.tokens {
			if now.After(entry.ExpiresAt) {
				delete(s.tokens, id)
			}
		}
		s.mu.Unlock()
	}
}
