package banking

import (
	"sync"
	"testing"

	"bank-server/internal/store"
	"bank-server/pkg/models"
)

// setupTestEnvironment initializes a fresh store and service for each test.
func setupTestEnvironment() (*store.MemoryStore, Service) {
	memStore := store.NewMemoryStore()
	svc := NewService(memStore)
	return memStore, svc
}

// defaultPassword is a helper to easily generate the [8]byte password
func defaultPassword() [8]byte {
	return [8]byte{'1', '2', '3', '4', '5', '6', '7', '8'}
}

func TestService_OpenAccount(t *testing.T) {
	_, svc := setupTestEnvironment()

	accNo := svc.OpenAccount("Alice", defaultPassword(), models.SGD, 1000.50)

	if accNo < 10000 {
		t.Errorf("Expected account number to be >= 10000, got %d", accNo)
	}
}

func TestService_Withdraw(t *testing.T) {
	_, svc := setupTestEnvironment()
	pw := defaultPassword()
	accNo := svc.OpenAccount("Bob", pw, models.USD, 500.0)

	// Table-driven tests for all S3 Withdrawal rules
	tests := []struct {
		name          string
		holderName    string
		accNo         uint32
		attemptPw     [8]byte
		currency      models.Currency
		amount        float64
		expectedErr   error
		expectBalance float64
	}{
		{"Valid Withdrawal", "Bob", accNo, pw, models.USD, 100.0, nil, 400.0},
		{"Wrong Password", "Bob", accNo, [8]byte{'w','r','o','n','g'}, models.USD, 50.0, ErrInvalidCredentials, 400.0},
		{"Wrong Name", "Eve", accNo, pw, models.USD, 50.0, ErrAccountMismatch, 400.0},
		{"Wrong Currency", "Bob", accNo, pw, models.SGD, 50.0, ErrCurrencyMismatch, 400.0},
		{"Insufficient Funds", "Bob", accNo, pw, models.USD, 1000.0, ErrInsufficientFunds, 400.0},
		{"Non-existent Account", "Bob", 99999, pw, models.USD, 50.0, ErrInvalidCredentials, 400.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newBal, err := svc.Withdraw(tt.holderName, tt.accNo, tt.attemptPw, tt.currency, tt.amount)

			if err != tt.expectedErr {
				t.Errorf("Expected error '%v', got '%v'", tt.expectedErr, err)
			}

			// If it was a successful transaction, verify the returned balance
			if err == nil && newBal != tt.expectBalance {
				t.Errorf("Expected balance %f, got %f", tt.expectBalance, newBal)
			}
		})
	}
}

func TestService_Deposit(t *testing.T) {
	_, svc := setupTestEnvironment()
	pw := defaultPassword()
	accNo := svc.OpenAccount("Charlie", pw, models.EUR, 100.0)

	tests := []struct {
		name          string
		holderName    string
		accNo         uint32
		attemptPw     [8]byte
		currency      models.Currency
		amount        float64
		expectedErr   error
	}{
		{"Valid Deposit", "Charlie", accNo, pw, models.EUR, 50.0, nil},
		{"Wrong Password", "Charlie", accNo, [8]byte{'0'}, models.EUR, 50.0, ErrInvalidCredentials},
		{"Wrong Currency", "Charlie", accNo, pw, models.USD, 50.0, ErrCurrencyMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Deposit(tt.holderName, tt.accNo, tt.attemptPw, tt.currency, tt.amount)
			if err != tt.expectedErr {
				t.Errorf("Expected error '%v', got '%v'", tt.expectedErr, err)
			}
		})
	}
}

func TestService_CloseAccount(t *testing.T) {
	memStore, svc := setupTestEnvironment()
	pw := defaultPassword()
	accNo := svc.OpenAccount("Dave", pw, models.SGD, 0.0)

	// Attempt invalid closes
	if err := svc.CloseAccount("Dave", accNo, [8]byte{'b','a','d'}); err != ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials on bad password, got %v", err)
	}

	// Valid close
	if err := svc.CloseAccount("Dave", accNo, pw); err != nil {
		t.Errorf("Expected no error on valid close, got %v", err)
	}

	if _, err := memStore.GetAccount(accNo); err != store.ErrAccountNotFound {
		t.Errorf("Expected account to be deleted from store")
	}
}

// TestService_ConcurrentWithdrawals PROVES your fine-grained mutexes work.
func TestService_ConcurrentWithdrawals(t *testing.T) {
	_, svc := setupTestEnvironment()
	pw := defaultPassword()
	
	// Create an account with exactly $50
	accNo := svc.OpenAccount("Eve", pw, models.SGD, 50.0)

	var wg sync.WaitGroup
	routines := 100

	// Fire off 100 simultaneous requests to withdraw $1
	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// We ignore the error here because we EXPECT 50 of them to fail 
			// with ErrInsufficientFunds once the balance hits 0.
			_, _ = svc.Withdraw("Eve", accNo, pw, models.SGD, 1.0)
		}()
	}

	wg.Wait() // Wait for all 100 goroutines to finish colliding

	// The balance should be exactly $0, not negative, and not > $0 (which would mean a lost update).
	// We do a deposit of $0 just to safely read the final balance via the service layer.
	finalBalance, _ := svc.Deposit("Eve", accNo, pw, models.SGD, 0.0)
	
	if finalBalance != 0.0 {
		t.Errorf("Expected final balance to be strictly 0.0 after concurrent overdraft attempts, got %f", finalBalance)
	}
}