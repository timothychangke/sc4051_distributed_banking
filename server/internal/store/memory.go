package store

import (
	"errors"
	"sync"
	"bank-server/pkg/models"
)

// ErrAccountNotFound is returned when an operation references a non-existent account ID.
var ErrAccountNotFound = errors.New("account not found")

// MemoryStore provides thread-safe, in-memory storage for bank accounts.
type MemoryStore struct {
	mu       sync.RWMutex
	accounts map[uint32]*models.Account
	nextID   uint32
}

// NewMemoryStore initializes and returns a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		accounts: make(map[uint32]*models.Account),
		nextID:   10000,
	}
}

// CreateAccount assigns a unique ID to the account, stores it safely, and returns the generated ID.
func (s *MemoryStore) CreateAccount(acc *models.Account) uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID
	acc.AccountNumber = id
	s.accounts[id] = acc
	s.nextID++

	return id
}

// GetAccount retrieves an account by its ID. 
// It returns ErrAccountNotFound if the account does not exist.
func (s *MemoryStore) GetAccount(id uint32) (*models.Account, error) {
	// The Read lock here suffices.
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, exists := s.accounts[id]
	if !exists {
		return nil, ErrAccountNotFound
	}

	return acc, nil
}

// UpdateAccount overwrites an existing account's data. 
// It returns ErrAccountNotFound if the target account is not present in the store.
func (s *MemoryStore) UpdateAccount(acc *models.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.accounts[acc.AccountNumber]; !exists {
		return ErrAccountNotFound
	}

	s.accounts[acc.AccountNumber] = acc
	return nil
}

// DeleteAccount removes an account from the store by its ID.
// It returns ErrAccountNotFound if the account does not exist prior to deletion.
func (s *MemoryStore) DeleteAccount(id uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.accounts[id]; !exists {
		return ErrAccountNotFound
	}

	delete(s.accounts, id)
	return nil
}