package models

// Currency represents a supported monetary unit.
// It is backed by a uint8 to satisfy manual binary marshalling requirements.
type Currency uint8

const (
	SGD Currency = iota + 1
	USD
	EUR
)

// Account represents the persistent state of a bank user.
type Account struct {
	AccountNumber uint32
	HolderName    string
	// Password must be exactly 8 bytes as per protocol specifications.
	Password      [8]byte
	CurrencyType  Currency
	Balance       float64
}