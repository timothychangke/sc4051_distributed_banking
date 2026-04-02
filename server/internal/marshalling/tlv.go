package marshal

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ─────────────────────────────────────────────────────────────────────
// TLV Decoder for the C++ client's CommandEncoder payload format.
//
// The C++ client encodes each field as:
//   [FieldID (1 byte)] [FieldLength (4 bytes, big-endian)] [FieldValue (N bytes)]
//
// Fields can arrive in ANY order and are OPTIONAL: only present fields
// are serialized. The decoder loops until there aren't enough bytes left
// for another TLV header (5 bytes = 1 tag + 4 length).
//
// This file is the single most important piece of cross-language glue.
// If anything here is wrong, every request from the C++ client fails.
// ─────────────────────────────────────────────────────────────────────

// ParsedCommand holds the fields extracted from a TLV-encoded request payload.
// Fields are pointers so we can distinguish "not present" (nil) from "zero value".
// For example, an account number of 0 is different from no account number at all.
type ParsedCommand struct {
	Service               *uint8
	AccountNumber         *uint32
	AccountOwnerName      *string
	AccountPassword       *string
	TxAccountNumber       *uint32
	TxAccountOwnerName    *string
	MonetaryValue         *float64
	Currency              *uint8
	MonitorUpdates        *string // callback content string; purpose TBD with C++ teammate
	MonitorTimeoutSeconds *uint32 // monitor interval in seconds
}

// DecodeTLV parses the TLV-encoded payload portion of a client request.
// The caller should pass ONLY the payload bytes (i.e., after stripping the
// 5-byte semantics header). This mirrors the C++ CommandEncoder::decode_message()
// loop exactly: same field IDs, same length checks, same byte order.
func DecodeTLV(payload []byte) (*ParsedCommand, error) {
	cmd := &ParsedCommand{}
	offset := 0

	for {
		// Need at least 5 bytes for the next TLV header (1 tag + 4 length).
		// If we don't have that many left, we're done: this is not an error,
		// it's how the C++ encoder signals "no more fields."
		if offset+TLVHeaderSize > len(payload) {
			break
		}

		// Read the field tag
		fieldID := payload[offset]
		offset++

		// Read the field length (4 bytes, big-endian)
		fieldLen := binary.BigEndian.Uint32(payload[offset : offset+4])
		offset += 4

		// Bounds check: does the value actually fit in the remaining bytes?
		if offset+int(fieldLen) > len(payload) {
			return nil, fmt.Errorf("field 0x%02X at offset %d: length %d overflows packet (remaining: %d)",
				fieldID, offset-TLVHeaderSize, fieldLen, len(payload)-offset)
		}

		// Slice out the value bytes for this field
		valueBytes := payload[offset : offset+int(fieldLen)]

		// Decode each field based on its tag. The expected sizes here match
		// what the C++ CommandEncoder writes: if a field has the wrong length,
		// something went very wrong on the wire.
		switch fieldID {
		case FieldService:
			if fieldLen != 1 {
				return nil, fmt.Errorf("field Service: expected 1 byte, got %d", fieldLen)
			}
			v := valueBytes[0]
			cmd.Service = &v

		case FieldAccountNumber:
			if fieldLen != 4 {
				return nil, fmt.Errorf("field AccountNumber: expected 4 bytes, got %d", fieldLen)
			}
			v := binary.BigEndian.Uint32(valueBytes)
			cmd.AccountNumber = &v

		case FieldAccountOwnerName:
			s := string(valueBytes)
			cmd.AccountOwnerName = &s

		case FieldAccountPassword:
			s := string(valueBytes)
			cmd.AccountPassword = &s

		case FieldTxAccountNumber:
			if fieldLen != 4 {
				return nil, fmt.Errorf("field TxAccountNumber: expected 4 bytes, got %d", fieldLen)
			}
			v := binary.BigEndian.Uint32(valueBytes)
			cmd.TxAccountNumber = &v

		case FieldTxAccountOwnerName:
			s := string(valueBytes)
			cmd.TxAccountOwnerName = &s

		case FieldMonetaryValue:
			if fieldLen != 8 {
				return nil, fmt.Errorf("field MonetaryValue: expected 8 bytes, got %d", fieldLen)
			}
			bits := binary.BigEndian.Uint64(valueBytes)
			v := math.Float64frombits(bits)
			cmd.MonetaryValue = &v

		case FieldCurrency:
			if fieldLen != 1 {
				return nil, fmt.Errorf("field Currency: expected 1 byte, got %d", fieldLen)
			}
			v := valueBytes[0]
			cmd.Currency = &v

		case FieldMonitorUpdates:
			s := string(valueBytes)
			cmd.MonitorUpdates = &s

		case FieldMonitorTimeoutSeconds:
			if fieldLen != 4 {
				return nil, fmt.Errorf("field MonitorTimeoutSeconds: expected 4 bytes, got %d", fieldLen)
			}
			v := binary.BigEndian.Uint32(valueBytes)
			cmd.MonitorTimeoutSeconds = &v

		default:
			return nil, fmt.Errorf("unknown field ID 0x%02X at offset %d", fieldID, offset-TLVHeaderSize)
		}

		offset += int(fieldLen)
	}

	return cmd, nil
}

// EncodeTLVFields encodes a set of TLV fields into a byte slice.
// This is used to build reply content that the C++ client can decode
// with its existing CommandEncoder::decode_message().
//
// Each field is written as [FieldID(1)][Length(4 BE)][Value(N)].
// The caller passes field entries which are appended sequentially.
func EncodeTLVFields(fields []TLVField) []byte {
	enc := NewEncoder()
	for _, f := range fields {
		enc.PutUint8(f.ID)
		enc.PutUint32(uint32(len(f.Value)))
		enc.PutBytes(f.Value)
	}
	return enc.Bytes()
}

// TLVField represents a single tag-length-value entry ready for encoding.
// The Value slice must already be in wire format (big-endian for integers).
type TLVField struct {
	ID    uint8
	Value []byte
}

// Convenience constructors for building TLV fields without manual byte juggling.
// These are the Go equivalents of the C++ CommandEncoder::encode_* functions.

// TLVUint8 creates a TLV field containing a single byte value.
func TLVUint8(fieldID uint8, v uint8) TLVField {
	return TLVField{ID: fieldID, Value: []byte{v}}
}

// TLVUint32 creates a TLV field containing a big-endian uint32.
func TLVUint32(fieldID uint8, v uint32) TLVField {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return TLVField{ID: fieldID, Value: b}
}

// TLVFloat64 creates a TLV field containing a big-endian IEEE 754 float64.
func TLVFloat64(fieldID uint8, v float64) TLVField {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(v))
	return TLVField{ID: fieldID, Value: b}
}

// TLVString creates a TLV field containing raw string bytes.
// The TLV length header handles the size: no separate length prefix inside the value.
func TLVString(fieldID uint8, v string) TLVField {
	return TLVField{ID: fieldID, Value: []byte(v)}
}
