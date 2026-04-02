package marshal

import (
	"bank-server/pkg/models"
)


// MsgTypeCallback is the message type byte for monitor push updates.
// Must stay in sync with protocol.MsgTypeCallback (0x02).
const MsgTypeCallback uint8 = 0x02

func MarshalCallbackUpdate(update models.AccountUpdate) ([]byte, error) {
	// Pre-calculate the total size so we allocate once.
	// 1 (MsgType) + 1 (ServiceID) + 4 (AccNo) + 4 (name len) + len(name) + 1 (Currency) + 8 (Balance)
	totalSize := 1 + 1 + 4 + 4 + len(update.HolderName) + 1 + 8
	enc := NewEncoderWithCap(totalSize)

	// Byte 0: Message type: 0x02 tells the client this is a callback, not a reply
	enc.PutUint8(MsgTypeCallback)

	// Byte 1: Which operation triggered this update (Open=1, Close=2, etc.)
	enc.PutUint8(update.ServiceID)

	// Bytes 2-5: The account that was mutated
	enc.PutUint32(update.AccountNumber)

	// Bytes 6-9 + N: Holder name as a length-prefixed string
	// [4 bytes big-endian length] [N bytes raw string data]
	enc.PutLengthPrefixedString(update.HolderName)

	// Next byte: Currency type (SGD=1, USD=2, EUR=3)
	enc.PutUint8(uint8(update.CurrencyType))

	// Last 8 bytes: New balance as big-endian IEEE 754 float64
	enc.PutFloat64(update.NewBalance)

	return enc.Bytes(), nil
}
