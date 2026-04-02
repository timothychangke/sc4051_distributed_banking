package marshal

// ─────────────────────────────────────────────────────────────────────
// Field IDs for the TLV (Tag-Length-Value) wire format.
//
// These MUST match the C++ client's Protocol::FieldID enum byte-for-byte.
// If even one is off, the cross-language TLV decoder silently misreads
// every field in the packet. Confirm values against protocol.h before
// shipping: the .cpp files only reference these by name, not by number.
// ─────────────────────────────────────────────────────────────────────
const (
	FieldService               uint8 = 0x01
	FieldAccountNumber         uint8 = 0x02
	FieldAccountOwnerName      uint8 = 0x03
	FieldAccountPassword       uint8 = 0x04
	FieldTxAccountNumber       uint8 = 0x05
	FieldTxAccountOwnerName    uint8 = 0x06
	FieldMonetaryValue         uint8 = 0x07
	FieldCurrency              uint8 = 0x08
	FieldMonitorUpdates        uint8 = 0x09 // variable-length string; added in client rev2
	FieldMonitorTimeoutSeconds uint8 = 0x0A // uint32 seconds; added in client rev2
)

// Every TLV field on the wire is prefixed with:
//
//	[1 byte FieldID] [4 bytes FieldLength (big-endian)]
//
// so the minimum overhead per field is 5 bytes before the value even starts.
const TLVHeaderSize = 5

// validFieldIDs is a quick lookup set so we can reject garbage field IDs
// at decode time without a long switch statement in the hot path.
var validFieldIDs = map[uint8]bool{
	FieldService:               true,
	FieldAccountNumber:         true,
	FieldAccountOwnerName:      true,
	FieldAccountPassword:       true,
	FieldTxAccountNumber:       true,
	FieldTxAccountOwnerName:    true,
	FieldMonetaryValue:         true,
	FieldCurrency:              true,
	FieldMonitorUpdates:        true,
	FieldMonitorTimeoutSeconds: true,
}

// IsValidFieldID checks whether a raw byte is a known TLV field tag.
func IsValidFieldID(id uint8) bool {
	return validFieldIDs[id]
}
