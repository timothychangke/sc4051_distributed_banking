package banking

import (
	"errors"

	"bank-server/internal/store"
	"bank-server/pkg/models"
)
// Sentinel Errors to reuse throughout the logic
var (
	ErrInvalidCredentials = errors.New("invalid account number or password")
	ErrAccountMismatch    = errors.New("account name does not match records")
	ErrCurrencyMismatch   = errors.New("currency type does not match account")
	ErrInsufficientFunds  = errors.New("insufficient balance for withdrawal")
)

// Service defines the core banking operations 
type Service interface {
	OpenAccount(name string, pw [8]byte, curr models.Currency, balance float64) uint32
	CloseAccount(name string, accNo uint32, pw [8]byte) error
	Deposit(name string, accNo uint32, pw [8]byte, curr models.Currency, amount float64) (float64, error)
	Withdraw(name string, accNo uint32, pw [8]byte, curr models.Currency, amount float64) (float64, error)
}

// Private service struct
type service struct {
	store *store.MemoryStore
}

// Acts as the constructor to inject the storage dependency
func NewService(s *store.MemoryStore) Service {
	return &service{store: s}
}

// Private helper function to check auth
func checkAuth(acc *models.Account, name string, pw [8]byte) error {
	if acc.Password != pw {
		return ErrInvalidCredentials
	}
	if acc.HolderName != name {
		return ErrAccountMismatch
	}
	return nil
}

// Initialises a new bank account and returns the generated account number
func (s *service) OpenAccount(name string, pw [8]byte, curr models.Currency, balance float64) uint32 {
	acc := &models.Account{
		HolderName:   name,
		Password:     pw,
		CurrencyType: curr,
		Balance:      balance,
	}
	return s.store.CreateAccount(acc)
}

// Checks that account exists and credentials are correct before deletion
func (s *service) CloseAccount(name string, accNo uint32, pw [8]byte) error {
	acc, err := s.store.GetAccount(accNo)
	if err != nil {
		return ErrInvalidCredentials
	}

	if err := checkAuth(acc, name, pw); err != nil {
		return err
	}

	// Lock the account before checking credentials or deleting to make sure
	// no one is trying to withdraw from it while we close it.
	acc.Mu.Lock()
	defer acc.Mu.Unlock()

	return s.store.DeleteAccount(accNo)
}

// Checks that account exists and credentials are correct before adding funds
func (s *service) Deposit(name string, accNo uint32, pw [8]byte, curr models.Currency, amount float64) (float64, error) {
	acc, err := s.store.GetAccount(accNo)
	if err != nil {
		return 0, ErrInvalidCredentials
	}

	if err := checkAuth(acc, name, pw); err != nil {
		return 0, err
	}

	// Lock account before depositing
	acc.Mu.Lock()
	defer acc.Mu.Unlock()

	if acc.CurrencyType != curr {
		return 0, ErrCurrencyMismatch
	}

	acc.Balance += amount
	
	if err := s.store.UpdateAccount(acc); err != nil {
		return 0, err
	}

	return acc.Balance, nil
}

// Checks that account exists, credentials are correct and that there is sufficient funds before withdrawing funds
func (s *service) Withdraw(name string, accNo uint32, pw [8]byte, curr models.Currency, amount float64) (float64, error) {
	acc, err := s.store.GetAccount(accNo)
	if err != nil {
		return 0, ErrInvalidCredentials
	}

	if err := checkAuth(acc, name, pw); err != nil {
		return 0, err
	}

	// Lock the account before withdrawing
	acc.Mu.Lock()
	defer acc.Mu.Unlock()

	if acc.CurrencyType != curr {
		return 0, ErrCurrencyMismatch
	}

	// This prevents account balance from going negative 
	if acc.Balance < amount {
		return 0, ErrInsufficientFunds
	}

	acc.Balance -= amount
	
	if err := s.store.UpdateAccount(acc); err != nil {
		return 0, err
	}

	return acc.Balance, nil
}