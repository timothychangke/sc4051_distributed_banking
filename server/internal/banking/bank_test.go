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
		{"Wrong Password", "Bob", accNo, [8]byte{'w', 'r', 'o', 'n', 'g'}, models.USD, 50.0, ErrInvalidCredentials, 400.0},
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
		name        string
		holderName  string
		accNo       uint32
		attemptPw   [8]byte
		currency    models.Currency
		amount      float64
		expectedErr error
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
	if err := svc.CloseAccount("Dave", accNo, [8]byte{'b', 'a', 'd'}); err != ErrInvalidCredentials {
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

// TestService_CheckBalance verifies the idempotent S5 operation.
func TestService_CheckBalance(t *testing.T) {
	_, svc := setupTestEnvironment()
	pw := defaultPassword()
	accNo := svc.OpenAccount("Frank", pw, models.SGD, 250.75)

	tests := []struct {
		name          string
		holderName    string
		accNo         uint32
		attemptPw     [8]byte
		expectedErr   error
		expectBalance float64
	}{
		{"Valid Check", "Frank", accNo, pw, nil, 250.75},
		{"Wrong Password", "Frank", accNo, [8]byte{'w', 'r', 'o', 'n', 'g'}, ErrInvalidCredentials, 0},
		{"Wrong Name", "NotFrank", accNo, pw, ErrAccountMismatch, 0},
		{"Non-existent Account", "Frank", 99999, pw, ErrInvalidCredentials, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bal, err := svc.CheckBalance(tt.holderName, tt.accNo, tt.attemptPw)
			if err != tt.expectedErr {
				t.Errorf("Expected error '%v', got '%v'", tt.expectedErr, err)
			}
			if err == nil && bal != tt.expectBalance {
				t.Errorf("Expected balance %f, got %f", tt.expectBalance, bal)
			}
		})
	}
}

// TestService_Transfer verifies the non-idempotent S6 operation logic.
func TestService_Transfer(t *testing.T) {
	_, svc := setupTestEnvironment()
	pw := defaultPassword()

	aliceAcc := svc.OpenAccount("Alice", pw, models.SGD, 500.0)
	bobAcc := svc.OpenAccount("Bob", pw, models.SGD, 100.0)
	charlieAcc := svc.OpenAccount("Charlie", pw, models.USD, 100.0)

	tests := []struct {
		name        string
		fromName    string
		fromAcc     uint32
		attemptPw   [8]byte
		toAcc       uint32
		amount      float64
		expectedErr error
	}{
		{"Valid Transfer", "Alice", aliceAcc, pw, bobAcc, 200.0, nil},
		{"Insufficient Funds", "Alice", aliceAcc, pw, bobAcc, 1000.0, ErrInsufficientFunds},
		{"Same Account Transfer", "Alice", aliceAcc, pw, aliceAcc, 50.0, ErrTransferSameAccount},
		{"Wrong Password", "Alice", aliceAcc, [8]byte{'b', 'a', 'd'}, bobAcc, 50.0, ErrInvalidCredentials},
		{"Wrong Sender Name", "NotAlice", aliceAcc, pw, bobAcc, 50.0, ErrAccountMismatch},
		{"Invalid Receiver", "Alice", aliceAcc, pw, 99999, 50.0, ErrAccountNotFound},
		{"Currency Mismatch", "Alice", aliceAcc, pw, charlieAcc, 50.0, ErrCurrencyMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Transfer(tt.fromName, tt.fromAcc, tt.attemptPw, tt.toAcc, tt.amount)
			if err != tt.expectedErr {
				t.Errorf("Expected error '%v', got '%v'", tt.expectedErr, err)
			}
		})
	}

	// Verify final balances after the single successful $200 transfer
	aliceBal, _ := svc.CheckBalance("Alice", aliceAcc, pw)
	bobBal, _ := svc.CheckBalance("Bob", bobAcc, pw)

	if aliceBal != 300.0 {
		t.Errorf("Expected Alice to have 300.0, got %f", aliceBal)
	}
	if bobBal != 300.0 {
		t.Errorf("Expected Bob to have 300.0, got %f", bobBal)
	}
}

// TestService_ConcurrentTransfers PROVES your deadlock prevention works.
func TestService_ConcurrentTransfers(t *testing.T) {
	_, svc := setupTestEnvironment()
	pw := defaultPassword()

	// Initialize both accounts with $1000
	acc1 := svc.OpenAccount("Account1", pw, models.SGD, 1000.0)
	acc2 := svc.OpenAccount("Account2", pw, models.SGD, 1000.0)

	var wg sync.WaitGroup
	routines := 50 // 50 transfers in each direction

	// 1. Fire 50 goroutines transferring $10 from Acc1 to Acc2
	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.Transfer("Account1", acc1, pw, acc2, 10.0)
		}()
	}

	// 2. Fire 50 goroutines transferring $10 from Acc2 to Acc1 SIMULTANEOUSLY
	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.Transfer("Account2", acc2, pw, acc1, 10.0)
		}()
	}

	wg.Wait() // If you didn't have lock-ordering, the test would freeze forever right here.

	// Both accounts sent $500 and received $500, so balances should be exactly $1000.
	bal1, _ := svc.CheckBalance("Account1", acc1, pw)
	bal2, _ := svc.CheckBalance("Account2", acc2, pw)

	if bal1 != 1000.0 || bal2 != 1000.0 {
		t.Errorf("Race condition detected! Expected both balances to be 1000.0. Got Acc1: %f, Acc2: %f", bal1, bal2)
	}
}

func TestService_ConcurrentCheckBalanceAndWithdraw(t *testing.T) {
	_, svc := setupTestEnvironment()
	pw := defaultPassword()

	// Initial balance of $1000
	accNo := svc.OpenAccount("Grace", pw, models.SGD, 1000.0)

	var wg sync.WaitGroup
	routines := 100

	// 1. Spawn 100 Writers: Each withdraws $5 concurrently
	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.Withdraw("Grace", accNo, pw, models.SGD, 5.0)
		}()
	}

	// 2. Spawn 100 Readers: Each checks the balance concurrently
	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// We don't care about the returned value here, we just want to force
			// the memory read to happen at the exact same time as the writes above.
			_, _ = svc.CheckBalance("Grace", accNo, pw)
		}()
	}

	wg.Wait() // Wait for all 200 goroutines to finish colliding

	// 3. Verify final state
	finalBal, err := svc.CheckBalance("Grace", accNo, pw)
	if err != nil {
		t.Fatalf("Unexpected error checking final balance: %v", err)
	}

	// We started with 1000, withdrew 5 exactly 100 times.
	// 1000 - (100 * 5) = 500
	expectedBal := 500.0
	if finalBal != expectedBal {
		t.Errorf("Expected final balance to be %f, got %f", expectedBal, finalBal)
	}
}
