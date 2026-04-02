package marshal

import (
	"encoding/binary"
	"math"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────
// TLV decoder tests.
//
// These tests manually construct byte buffers that match what the C++
// CommandEncoder produces. If these pass, we can be confident that
// real packets from the C++ client will decode correctly.
// ─────────────────────────────────────────────────────────────────────

// buildTLVField is a test helper that constructs a single TLV entry
// exactly the way the C++ CommandEncoder does:
//
//	[FieldID (1 byte)] [FieldLength (4 bytes, BE)] [Value (N bytes)]
func buildTLVField(fieldID uint8, value []byte) []byte {
	buf := make([]byte, 0, 1+4+len(value))
	buf = append(buf, fieldID)

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(value)))
	buf = append(buf, lenBuf...)
	buf = append(buf, value...)

	return buf
}

func buildTLVUint8(fieldID uint8, v uint8) []byte {
	return buildTLVField(fieldID, []byte{v})
}

func buildTLVUint32(fieldID uint8, v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return buildTLVField(fieldID, b)
}

func buildTLVFloat64(fieldID uint8, v float64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(v))
	return buildTLVField(fieldID, b)
}

func buildTLVString(fieldID uint8, s string) []byte {
	return buildTLVField(fieldID, []byte(s))
}

// --- DecodeTLV tests ---

func TestDecodeTLV_EmptyPayload(t *testing.T) {
	cmd, err := DecodeTLV([]byte{})
	if err != nil {
		t.Fatalf("unexpected error on empty payload: %v", err)
	}
	// All fields should be nil since nothing was encoded
	if cmd.Service != nil || cmd.AccountNumber != nil || cmd.AccountOwnerName != nil {
		t.Error("expected all fields to be nil for empty payload")
	}
}

func TestDecodeTLV_SingleFieldService(t *testing.T) {
	payload := buildTLVUint8(FieldService, 3) // ServiceDeposit = 3

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Service == nil {
		t.Fatal("expected Service to be present")
	}
	if *cmd.Service != 3 {
		t.Errorf("expected Service=3, got %d", *cmd.Service)
	}
}

func TestDecodeTLV_SingleFieldAccountNumber(t *testing.T) {
	payload := buildTLVUint32(FieldAccountNumber, 10042)

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.AccountNumber == nil {
		t.Fatal("expected AccountNumber to be present")
	}
	if *cmd.AccountNumber != 10042 {
		t.Errorf("expected AccountNumber=10042, got %d", *cmd.AccountNumber)
	}
}

func TestDecodeTLV_SingleFieldMonetaryValue(t *testing.T) {
	payload := buildTLVFloat64(FieldMonetaryValue, 1234.56)

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.MonetaryValue == nil {
		t.Fatal("expected MonetaryValue to be present")
	}
	if *cmd.MonetaryValue != 1234.56 {
		t.Errorf("expected MonetaryValue=1234.56, got %f", *cmd.MonetaryValue)
	}
}

func TestDecodeTLV_StringFields(t *testing.T) {
	var payload []byte
	payload = append(payload, buildTLVString(FieldAccountOwnerName, "Alice")...)
	payload = append(payload, buildTLVString(FieldAccountPassword, "pass1234")...)

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.AccountOwnerName == nil || *cmd.AccountOwnerName != "Alice" {
		t.Errorf("expected AccountOwnerName='Alice', got %v", cmd.AccountOwnerName)
	}
	if cmd.AccountPassword == nil || *cmd.AccountPassword != "pass1234" {
		t.Errorf("expected AccountPassword='pass1234', got %v", cmd.AccountPassword)
	}
}

func TestDecodeTLV_FullOpenAccountRequest(t *testing.T) {
	// Simulate exactly what the C++ client sends for an Open Account request:
	// Service=1, Name="Bob", Password="secret12", Currency=1(SGD), Balance=500.0
	var payload []byte
	payload = append(payload, buildTLVUint8(FieldService, 1)...)
	payload = append(payload, buildTLVString(FieldAccountOwnerName, "Bob")...)
	payload = append(payload, buildTLVString(FieldAccountPassword, "secret12")...)
	payload = append(payload, buildTLVUint8(FieldCurrency, 1)...)
	payload = append(payload, buildTLVFloat64(FieldMonetaryValue, 500.0)...)

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *cmd.Service != 1 {
		t.Errorf("Service: want 1, got %d", *cmd.Service)
	}
	if *cmd.AccountOwnerName != "Bob" {
		t.Errorf("Name: want 'Bob', got '%s'", *cmd.AccountOwnerName)
	}
	if *cmd.AccountPassword != "secret12" {
		t.Errorf("Password: want 'secret12', got '%s'", *cmd.AccountPassword)
	}
	if *cmd.Currency != 1 {
		t.Errorf("Currency: want 1, got %d", *cmd.Currency)
	}
	if *cmd.MonetaryValue != 500.0 {
		t.Errorf("Balance: want 500.0, got %f", *cmd.MonetaryValue)
	}
	// Fields that weren't sent should remain nil
	if cmd.AccountNumber != nil {
		t.Error("AccountNumber should be nil for Open request")
	}
}

func TestDecodeTLV_FieldsInReverseOrder(t *testing.T) {
	// The C++ iterate() walks fields in declaration order, but our decoder
	// must handle ANY order. This test verifies that by sending fields
	// in the opposite order from what the C++ client typically sends.
	var payload []byte
	payload = append(payload, buildTLVFloat64(FieldMonetaryValue, 100.50)...)
	payload = append(payload, buildTLVUint8(FieldCurrency, 2)...)
	payload = append(payload, buildTLVString(FieldAccountPassword, "mypass")...)
	payload = append(payload, buildTLVUint32(FieldAccountNumber, 10001)...)
	payload = append(payload, buildTLVString(FieldAccountOwnerName, "Charlie")...)
	payload = append(payload, buildTLVUint8(FieldService, 3)...)

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *cmd.Service != 3 {
		t.Errorf("Service: want 3, got %d", *cmd.Service)
	}
	if *cmd.AccountOwnerName != "Charlie" {
		t.Errorf("Name: want 'Charlie', got '%s'", *cmd.AccountOwnerName)
	}
	if *cmd.AccountNumber != 10001 {
		t.Errorf("AccountNumber: want 10001, got %d", *cmd.AccountNumber)
	}
	if *cmd.MonetaryValue != 100.50 {
		t.Errorf("MonetaryValue: want 100.50, got %f", *cmd.MonetaryValue)
	}
}

func TestDecodeTLV_TransferRequest(t *testing.T) {
	// Transfer needs both source and destination account fields
	var payload []byte
	payload = append(payload, buildTLVUint8(FieldService, 7)...)
	payload = append(payload, buildTLVString(FieldAccountOwnerName, "Alice")...)
	payload = append(payload, buildTLVUint32(FieldAccountNumber, 10001)...)
	payload = append(payload, buildTLVString(FieldAccountPassword, "alicepass")...)
	payload = append(payload, buildTLVUint32(FieldTxAccountNumber, 10002)...)
	payload = append(payload, buildTLVFloat64(FieldMonetaryValue, 250.75)...)

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *cmd.AccountNumber != 10001 {
		t.Errorf("AccountNumber: want 10001, got %d", *cmd.AccountNumber)
	}
	if *cmd.TxAccountNumber != 10002 {
		t.Errorf("TxAccountNumber: want 10002, got %d", *cmd.TxAccountNumber)
	}
	if *cmd.MonetaryValue != 250.75 {
		t.Errorf("MonetaryValue: want 250.75, got %f", *cmd.MonetaryValue)
	}
}

func TestDecodeTLV_MonitorRegistration(t *testing.T) {
	// The new client sends monitor timeout as a TLV field, not flat bytes
	var payload []byte
	payload = append(payload, buildTLVUint8(FieldService, 5)...)
	payload = append(payload, buildTLVUint32(FieldMonitorTimeoutSeconds, 60)...)
	payload = append(payload, buildTLVString(FieldMonitorUpdates, "some_filter")...)

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd.MonitorTimeoutSeconds == nil || *cmd.MonitorTimeoutSeconds != 60 {
		t.Errorf("MonitorTimeoutSeconds: want 60, got %v", cmd.MonitorTimeoutSeconds)
	}
	if cmd.MonitorUpdates == nil || *cmd.MonitorUpdates != "some_filter" {
		t.Errorf("MonitorUpdates: want 'some_filter', got %v", cmd.MonitorUpdates)
	}
}

func TestDecodeTLV_UnknownFieldID(t *testing.T) {
	// A field ID that doesn't exist in our enum should produce an error
	payload := buildTLVUint8(0xFF, 42)

	_, err := DecodeTLV(payload)
	if err == nil {
		t.Fatal("expected error for unknown field ID 0xFF, got nil")
	}
}

func TestDecodeTLV_TruncatedFieldValue(t *testing.T) {
	// Construct a TLV header that claims 4 bytes but only has 2
	buf := []byte{FieldAccountNumber}
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, 4) // claims 4 bytes of value
	buf = append(buf, lenBuf...)
	buf = append(buf, 0x00, 0x01) // but only 2 bytes follow

	_, err := DecodeTLV(buf)
	if err == nil {
		t.Fatal("expected error for truncated field, got nil")
	}
}

func TestDecodeTLV_WrongLengthForFixedField(t *testing.T) {
	// AccountNumber should be exactly 4 bytes. Sending 2 should fail.
	badValue := make([]byte, 2)
	payload := buildTLVField(FieldAccountNumber, badValue)

	_, err := DecodeTLV(payload)
	if err == nil {
		t.Fatal("expected error for AccountNumber with wrong length")
	}
}

func TestDecodeTLV_EmptyString(t *testing.T) {
	// Edge case: a string field with zero length is valid (empty name)
	payload := buildTLVString(FieldAccountOwnerName, "")

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("unexpected error for empty string: %v", err)
	}
	if cmd.AccountOwnerName == nil || *cmd.AccountOwnerName != "" {
		t.Errorf("expected empty string, got %v", cmd.AccountOwnerName)
	}
}

func TestDecodeTLV_Float64SpecialValues(t *testing.T) {
	// Make sure edge-case floats round-trip correctly
	tests := []struct {
		name string
		val  float64
	}{
		{"zero", 0.0},
		{"negative", -1234.56},
		{"very small", 0.001},
		{"large", 999999999.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := buildTLVFloat64(FieldMonetaryValue, tt.val)
			cmd, err := DecodeTLV(payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd.MonetaryValue == nil || *cmd.MonetaryValue != tt.val {
				t.Errorf("want %f, got %v", tt.val, cmd.MonetaryValue)
			}
		})
	}
}

func TestDecodeTLV_IncompleteTLVHeader(t *testing.T) {
	// Only 3 bytes: not enough for a full TLV header (need 5).
	// This should NOT be an error; it means "no more fields."
	payload := []byte{0x01, 0x00, 0x00}

	cmd, err := DecodeTLV(payload)
	if err != nil {
		t.Fatalf("incomplete TLV header should stop cleanly, got error: %v", err)
	}
	// No fields should have been decoded
	if cmd.Service != nil {
		t.Error("expected nil Service when header was incomplete")
	}
}

// --- EncodeTLVFields tests ---

func TestEncodeTLVFields_SingleUint32(t *testing.T) {
	fields := []TLVField{TLVUint32(FieldAccountNumber, 10042)}
	encoded := EncodeTLVFields(fields)

	// Verify: decode it back and make sure we get the right value
	cmd, err := DecodeTLV(encoded)
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	if cmd.AccountNumber == nil || *cmd.AccountNumber != 10042 {
		t.Errorf("round-trip: want 10042, got %v", cmd.AccountNumber)
	}
}

func TestEncodeTLVFields_SingleFloat64(t *testing.T) {
	fields := []TLVField{TLVFloat64(FieldMonetaryValue, 777.88)}
	encoded := EncodeTLVFields(fields)

	cmd, err := DecodeTLV(encoded)
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	if cmd.MonetaryValue == nil || *cmd.MonetaryValue != 777.88 {
		t.Errorf("round-trip: want 777.88, got %v", cmd.MonetaryValue)
	}
}

func TestEncodeTLVFields_MultipleFields(t *testing.T) {
	fields := []TLVField{
		TLVUint8(FieldService, 1),
		TLVString(FieldAccountOwnerName, "Dave"),
		TLVUint32(FieldAccountNumber, 10099),
		TLVFloat64(FieldMonetaryValue, 42.0),
	}
	encoded := EncodeTLVFields(fields)

	cmd, err := DecodeTLV(encoded)
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	if *cmd.Service != 1 {
		t.Errorf("Service: want 1, got %d", *cmd.Service)
	}
	if *cmd.AccountOwnerName != "Dave" {
		t.Errorf("Name: want 'Dave', got '%s'", *cmd.AccountOwnerName)
	}
	if *cmd.AccountNumber != 10099 {
		t.Errorf("AccountNumber: want 10099, got %d", *cmd.AccountNumber)
	}
	if *cmd.MonetaryValue != 42.0 {
		t.Errorf("MonetaryValue: want 42.0, got %f", *cmd.MonetaryValue)
	}
}

func TestEncodeTLVFields_EmptyList(t *testing.T) {
	encoded := EncodeTLVFields(nil)
	if len(encoded) != 0 {
		t.Errorf("empty field list should produce 0 bytes, got %d", len(encoded))
	}
}

// --- TLV convenience constructor tests ---

func TestTLVUint8_ByteLayout(t *testing.T) {
	f := TLVUint8(FieldCurrency, 2)
	if f.ID != FieldCurrency {
		t.Errorf("ID: want 0x%02X, got 0x%02X", FieldCurrency, f.ID)
	}
	if len(f.Value) != 1 || f.Value[0] != 2 {
		t.Errorf("Value: want [2], got %v", f.Value)
	}
}

func TestTLVUint32_ByteLayout(t *testing.T) {
	f := TLVUint32(FieldAccountNumber, 0x0000FFFF)
	if len(f.Value) != 4 {
		t.Fatalf("Value length: want 4, got %d", len(f.Value))
	}
	got := binary.BigEndian.Uint32(f.Value)
	if got != 0x0000FFFF {
		t.Errorf("Value: want 0x0000FFFF, got 0x%08X", got)
	}
}

func TestTLVFloat64_ByteLayout(t *testing.T) {
	f := TLVFloat64(FieldMonetaryValue, 100.50)
	if len(f.Value) != 8 {
		t.Fatalf("Value length: want 8, got %d", len(f.Value))
	}
	bits := binary.BigEndian.Uint64(f.Value)
	got := math.Float64frombits(bits)
	if got != 100.50 {
		t.Errorf("Value: want 100.50, got %f", got)
	}
}

func TestTLVString_ByteLayout(t *testing.T) {
	f := TLVString(FieldAccountOwnerName, "Eve")
	if f.ID != FieldAccountOwnerName {
		t.Errorf("ID: want 0x%02X, got 0x%02X", FieldAccountOwnerName, f.ID)
	}
	if string(f.Value) != "Eve" {
		t.Errorf("Value: want 'Eve', got '%s'", string(f.Value))
	}
}
