// Defines the constants for the protocol used by the server and client to communicate.
package protocol

const (
	ServiceOpenAccount   = 1
	ServiceCloseAccount  = 2
	ServiceDeposit       = 3
	ServiceWithdraw      = 4
	ServiceMonitor       = 5
	ServiceGetBalance    = 6 // Op5: Idempotent
	ServiceTransferFunds = 7 // Op6: Non-idempotent
)
