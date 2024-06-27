package github

import (
	"sync"
)

type TokenManager struct {
	tokens    []string
	index     int
	exhausted bool
	mu        sync.Mutex
}

func NewTokenManager(tokens []string) *TokenManager {
	return &TokenManager{
		tokens:    tokens,
		index:     0,
		exhausted: false,
		mu:        sync.Mutex{},
	}
}

func (tm *TokenManager) GetToken() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.exhausted {
		return ""
	}
	token := tm.tokens[tm.index]
	return token
}

func (tm *TokenManager) IsExhausted() bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.exhausted
}

func (tm *TokenManager) RotateToken() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.index = (tm.index + 1) % len(tm.tokens)
	if tm.index == 0 {
		tm.exhausted = true
	}
}

func (tm *TokenManager) ResetExhaustion() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.exhausted = false
}
