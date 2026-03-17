package models

// Struct holds info of what changed
type AccountUpdate struct {
	// Which service triggered this update (Open=1, Close=2, Deposit=3, Withdraw=4, Transfer=7)
	ServiceID uint8

	// The account that was mutated
	AccountNumber uint32
	HolderName    string
	CurrencyType  Currency
	NewBalance    float64
}