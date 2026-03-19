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


// Message types.
// This is the very first byte of every server-to-client packet.
// It lets the client know whether its reading a normal reply
// or a callback push without parsing the rest of the payload.
const (
	MsgTypeReply    uint8 = 0x01 // Standard request response reply
	MsgTypeCallback uint8 = 0x02 // Push from the monitor system
)
 
// Status codes that match that of the client
// Would be nicer to have one unified file for this but this would do for now
const (
	StatusSuccess          uint8 = 0
	StatusErrAccNotFound   uint8 = 1
	StatusErrInvalidCreds  uint8 = 2
	StatusErrAccMismatch   uint8 = 3
	StatusErrCurrMismatch  uint8 = 4
	StatusErrInsuffFunds   uint8 = 5
	StatusErrSameAccount   uint8 = 6
	StatusErrInternal      uint8 = 7
)