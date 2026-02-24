package store

import (
	"sync"
	"testing"
	"bank-server/pkg/models"
)

func TestMemoryStore_CreateAndGet(t *testing.T) {
	s := NewMemoryStore()

	acc := &models.Account{
		HolderName: "Alice",
		Balance:    100.50,
	}

	id := s.CreateAccount(acc)
	if id != 10000 {
		t.Errorf("Expected first ID to be 10000, got %d", id)
	}

	retrieved, err := s.GetAccount(id)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if retrieved.HolderName != "Alice" {
		t.Errorf("Expected HolderName 'Alice', got '%s'", retrieved.HolderName)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	s := NewMemoryStore()

	_, err := s.GetAccount(99999)
	if err != ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestMemoryStore_Update(t *testing.T) {
	s := NewMemoryStore()
	id := s.CreateAccount(&models.Account{HolderName: "Bob", Balance: 50.0})

	// Happy path
	updatedAcc := &models.Account{AccountNumber: id, HolderName: "Bob Updated", Balance: 75.0}
	err := s.UpdateAccount(updatedAcc)
	if err != nil {
		t.Fatalf("Unexpected error on update: %v", err)
	}

	retrieved, _ := s.GetAccount(id)
	if retrieved.Balance != 75.0 {
		t.Errorf("Expected balance 75.0, got %f", retrieved.Balance)
	}

	// Sad path
	err = s.UpdateAccount(&models.Account{AccountNumber: 99999})
	if err != ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound for invalid update, got %v", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	id := s.CreateAccount(&models.Account{HolderName: "Charlie"})

	// Happy path
	err := s.DeleteAccount(id)
	if err != nil {
		t.Fatalf("Unexpected error on delete: %v", err)
	}

	_, err = s.GetAccount(id)
	if err != ErrAccountNotFound {
		t.Errorf("Expected account to be deleted, but got err: %v", err)
	}

	// Sad path
	err = s.DeleteAccount(99999)
	if err != ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound for invalid delete, got %v", err)
	}
}

// TestMemoryStore_Concurrency ensures the RWMutex safely handles highly concurrent access
// without causing panics or data races.
func TestMemoryStore_Concurrency(t *testing.T) {
	s := NewMemoryStore()
	var wg sync.WaitGroup
	workers := 100

	// Concurrent Creates
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			s.CreateAccount(&models.Account{Balance: float64(val)})
		}(i)
	}
	wg.Wait()

	// Verify we generated exactly 100 accounts without losing any due to race conditions
	if len(s.accounts) != workers {
		t.Errorf("Expected %d accounts, got %d", workers, len(s.accounts))
	}

	// Concurrent Reads
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id uint32) {
			defer wg.Done()
			_, _ = s.GetAccount(id)
		}(uint32(10000 + i))
	}
	wg.Wait()
}