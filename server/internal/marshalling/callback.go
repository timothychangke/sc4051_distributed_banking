package marshal

import (
	"bank-server/pkg/models"
)

// ─────────────────────────────────────────────────────────────────────
// Callback packet builder for the monitor system.
//
// When a mutation happens (Open, Close, Deposit, Withdraw, Transfer),
// the monitor manager pushes an update to all subscribed clients. These
// callback packets use a FLAT layout (no TLV) with MsgTypeCallback (0x02)
// so the C++ client can distinguish them from normal replies at the first byte.
//
// Wire layout:
// ┌──────────┬──────────┬──────────────┬───────────────────┬──────────┬──────────┐
// │ MsgType  │ ServiceID│ AccountNumber│ HolderName        │ Currency │ Balance  │
// │ (1 byte) │ (1 byte) │ (4 bytes BE) │ [4B len][N data]  │ (1 byte) │ (8B BE)  │
// │ 0x02     │          │              │                   │          │ float64  │
// └──────────┴──────────┴──────────────┴───────────────────┴──────────┴──────────┘
//
// Minimum size (empty name): 1 + 1 + 4 + 4 + 0 + 1 + 8 = 19 bytes
// ─────────────────────────────────────────────────────────────────────

// MsgTypeCallback is the message type byte for monitor push updates.
// Must stay in sync with protocol.MsgTypeCallback (0x02).
const MsgTypeCallback uint8 = 0x02

// MarshalCallbackUpdate converts an AccountUpdate into the raw bytes
// that get pushed to monitoring clients over UDP. This function is
// injected into monitor.NewManager() at startup so the monitor system
// doesn't need to know anything about wire formats.
//
// The layout is flat and fixed-order (not TLV) because:
//   - Callback packets are fire-and-forget, no request/reply matching needed
//   - The C++ client's monitor loop expects a specific byte sequence
//   - TLV overhead is unnecessary when every callback has the same fields
func MarshalCallbackUpdate(update models.AccountUpdate) ([]byte, error) {
	// Pre-calculate the total size so we allocate once.
	// 1 (MsgType) + 1 (ServiceID) + 4 (AccNo) + 4 (name len) + len(name) + 1 (Currency) + 8 (Balance)
	totalSize := 1 + 1 + 4 + 4 + len(update.HolderName) + 1 + 8
	enc := NewEncoderWithCap(totalSize)

	// Byte 0: Message type — 0x02 tells the client this is a callback, not a reply
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