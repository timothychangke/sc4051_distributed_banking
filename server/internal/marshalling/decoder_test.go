package marshal

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────
// Happy path: read primitives from known byte sequences
// ─────────────────────────────────────────────────────────────────────

func TestReadUint8(t *testing.T) {
	dec := NewDecoder([]byte{0x00, 0xFF, 0x42})

	tests := []uint8{0x00, 0xFF, 0x42}
	for i, want := range tests {
		got, err := dec.ReadUint8()
		if err != nil {
			t.Fatalf("ReadUint8 #%d: unexpected error: %v", i, err)
		}
		if got != want {
			t.Errorf("ReadUint8 #%d = 0x%02X, want 0x%02X", i, got, want)
		}
	}
}

func TestReadUint16(t *testing.T) {
	buf := make([]byte, 6)
	binary.BigEndian.PutUint16(buf[0:2], 0)
	binary.BigEndian.PutUint16(buf[2:4], 1234)
	binary.BigEndian.PutUint16(buf[4:6], 0xFFFF)

	dec := NewDecoder(buf)

	tests := []uint16{0, 1234, 0xFFFF}
	for i, want := range tests {
		got, err := dec.ReadUint16()
		if err != nil {
			t.Fatalf("ReadUint16 #%d: unexpected error: %v", i, err)
		}
		if got != want {
			t.Errorf("ReadUint16 #%d = %d, want %d", i, got, want)
		}
	}
}

func TestReadUint32(t *testing.T) {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], 0)
	binary.BigEndian.PutUint32(buf[4:8], 10000)
	binary.BigEndian.PutUint32(buf[8:12], 0xFFFFFFFF)

	dec := NewDecoder(buf)

	tests := []uint32{0, 10000, 0xFFFFFFFF}
	for i, want := range tests {
		got, err := dec.ReadUint32()
		if err != nil {
			t.Fatalf("ReadUint32 #%d: unexpected error: %v", i, err)
		}
		if got != want {
			t.Errorf("ReadUint32 #%d = %d, want %d", i, got, want)
		}
	}
}

func TestReadFloat64(t *testing.T) {
	values := []float64{0.0, 100.50, -500.25, 1234.56, 999999999.99}

	for _, want := range values {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, math.Float64bits(want))

		dec := NewDecoder(buf)
		got, err := dec.ReadFloat64()
		if err != nil {
			t.Fatalf("ReadFloat64(%v): unexpected error: %v", want, err)
		}
		if got != want {
			t.Errorf("ReadFloat64 = %v, want %v", got, want)
		}
	}
}

func TestReadFloat64_KnownWireBytes(t *testing.T) {
	// The exact bytes for 100.50 in big-endian IEEE 754.
	// Cross-check this against a hex dump from the C++ client to verify
	// the two sides agree on float encoding.
	wire := []byte{0x40, 0x59, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00}

	dec := NewDecoder(wire)
	got, err := dec.ReadFloat64()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 100.50 {
		t.Errorf("ReadFloat64 = %v, want 100.50", got)
	}
}

func TestReadString(t *testing.T) {
	dec := NewDecoder([]byte("Alice"))

	got, err := dec.ReadString(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Alice" {
		t.Errorf("ReadString = %q, want %q", got, "Alice")
	}
}

func TestReadString_Empty(t *testing.T) {
	dec := NewDecoder([]byte{})

	got, err := dec.ReadString(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("ReadString(0) = %q, want empty", got)
	}
}

func TestReadBytes(t *testing.T) {
	input := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	dec := NewDecoder(input)

	got, err := dec.ReadBytes(4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertBytesEqual(t, got, input)

	// Verify it's a copy — mutating the result shouldn't affect the decoder
	got[0] = 0x00
	if dec.buf[0] != 0xDE {
		t.Error("ReadBytes returned a reference to the internal buffer, not a copy")
	}
}

func TestReadBytes_Empty(t *testing.T) {
	dec := NewDecoder([]byte{0x01})

	got, err := dec.ReadBytes(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ReadBytes(0) returned %d bytes, want 0", len(got))
	}
}

// ─────────────────────────────────────────────────────────────────────
// Offset tracking and helpers
// ─────────────────────────────────────────────────────────────────────

func TestOffset(t *testing.T) {
	dec := NewDecoder(make([]byte, 20))

	if dec.Offset() != 0 {
		t.Errorf("initial Offset() = %d, want 0", dec.Offset())
	}

	dec.ReadUint8()  // +1
	dec.ReadUint32() // +4
	if dec.Offset() != 5 {
		t.Errorf("after reading 5 bytes, Offset() = %d, want 5", dec.Offset())
	}
}

func TestRemaining(t *testing.T) {
	dec := NewDecoder(make([]byte, 10))

	if dec.Remaining() != 10 {
		t.Errorf("initial Remaining() = %d, want 10", dec.Remaining())
	}

	dec.ReadUint32() // consumes 4
	if dec.Remaining() != 6 {
		t.Errorf("after ReadUint32, Remaining() = %d, want 6", dec.Remaining())
	}
}

func TestSkip(t *testing.T) {
	// Simulate skipping the 5-byte semantics header to get to TLV payload
	buf := []byte{
		0x01,                   // ServiceID
		0x00, 0x00, 0x00, 0x2A, // RequestID = 42
		0x03,                   // first TLV field tag (FieldAccountOwnerName)
		0x00, 0x00, 0x00, 0x03, // TLV length = 3
		'B', 'o', 'b', // "Bob"
	}

	dec := NewDecoder(buf)

	// Skip the 5-byte header
	if err := dec.Skip(5); err != nil {
		t.Fatalf("Skip(5): %v", err)
	}

	if dec.Offset() != 5 {
		t.Errorf("after Skip(5), Offset() = %d, want 5", dec.Offset())
	}

	// Now we should be able to read the TLV field tag
	tag, err := dec.ReadUint8()
	if err != nil {
		t.Fatalf("ReadUint8: %v", err)
	}
	if tag != FieldAccountOwnerName {
		t.Errorf("tag = 0x%02X, want 0x%02X", tag, FieldAccountOwnerName)
	}
}

func TestSkip_Zero(t *testing.T) {
	dec := NewDecoder([]byte{0x01})
	if err := dec.Skip(0); err != nil {
		t.Errorf("Skip(0) returned unexpected error: %v", err)
	}
	if dec.Offset() != 0 {
		t.Errorf("after Skip(0), Offset() = %d, want 0", dec.Offset())
	}
}

// ─────────────────────────────────────────────────────────────────────
// Error cases: every Read must fail cleanly on underflow
// ─────────────────────────────────────────────────────────────────────

func TestReadUint8_Underflow(t *testing.T) {
	dec := NewDecoder([]byte{})
	_, err := dec.ReadUint8()
	if err == nil {
		t.Fatal("expected error on empty buffer, got nil")
	}
	if !errors.Is(err, ErrBufferUnderflow) {
		t.Errorf("expected ErrBufferUnderflow, got: %v", err)
	}
}

func TestReadUint16_Underflow(t *testing.T) {
	// Only 1 byte available, need 2
	dec := NewDecoder([]byte{0x01})
	_, err := dec.ReadUint16()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrBufferUnderflow) {
		t.Errorf("expected ErrBufferUnderflow, got: %v", err)
	}
}

func TestReadUint32_Underflow(t *testing.T) {
	// Only 3 bytes available, need 4
	dec := NewDecoder([]byte{0x01, 0x02, 0x03})
	_, err := dec.ReadUint32()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrBufferUnderflow) {
		t.Errorf("expected ErrBufferUnderflow, got: %v", err)
	}
}

func TestReadFloat64_Underflow(t *testing.T) {
	// Only 7 bytes available, need 8
	dec := NewDecoder(make([]byte, 7))
	_, err := dec.ReadFloat64()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrBufferUnderflow) {
		t.Errorf("expected ErrBufferUnderflow, got: %v", err)
	}
}

func TestReadString_Underflow(t *testing.T) {
	dec := NewDecoder([]byte("Hi"))
	_, err := dec.ReadString(10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrBufferUnderflow) {
		t.Errorf("expected ErrBufferUnderflow, got: %v", err)
	}
}

func TestReadString_NegativeLength(t *testing.T) {
	dec := NewDecoder([]byte{0x01})
	_, err := dec.ReadString(-1)
	if err == nil {
		t.Fatal("expected error on negative length, got nil")
	}
}

func TestReadBytes_Underflow(t *testing.T) {
	dec := NewDecoder([]byte{0x01})
	_, err := dec.ReadBytes(5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrBufferUnderflow) {
		t.Errorf("expected ErrBufferUnderflow, got: %v", err)
	}
}

func TestReadBytes_NegativeLength(t *testing.T) {
	dec := NewDecoder([]byte{0x01})
	_, err := dec.ReadBytes(-1)
	if err == nil {
		t.Fatal("expected error on negative length, got nil")
	}
}

func TestSkip_Underflow(t *testing.T) {
	dec := NewDecoder([]byte{0x01, 0x02})
	err := dec.Skip(10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrBufferUnderflow) {
		t.Errorf("expected ErrBufferUnderflow, got: %v", err)
	}
}

func TestSkip_NegativeCount(t *testing.T) {
	dec := NewDecoder([]byte{0x01})
	err := dec.Skip(-1)
	if err == nil {
		t.Fatal("expected error on negative skip, got nil")
	}
}

// ─────────────────────────────────────────────────────────────────────
// Underflow doesn't corrupt state: offset should stay where it was
// ─────────────────────────────────────────────────────────────────────

func TestUnderflowDoesNotAdvanceOffset(t *testing.T) {
	dec := NewDecoder([]byte{0x01})

	// Read the one available byte
	dec.ReadUint8()
	offsetBefore := dec.Offset()

	// Now every read should fail — and the offset shouldn't move
	dec.ReadUint8()
	dec.ReadUint16()
	dec.ReadUint32()
	dec.ReadFloat64()
	dec.ReadString(1)
	dec.ReadBytes(1)
	dec.Skip(1)

	if dec.Offset() != offsetBefore {
		t.Errorf("offset changed from %d to %d after failed reads", offsetBefore, dec.Offset())
	}
}

// ─────────────────────────────────────────────────────────────────────
// Round-trip: Encoder → Decoder should produce the original values
// ─────────────────────────────────────────────────────────────────────

func TestRoundTrip_AllPrimitives(t *testing.T) {
	// Encode
	enc := NewEncoder()
	enc.PutUint8(0x42)
	enc.PutUint16(9999)
	enc.PutUint32(123456789)
	enc.PutFloat64(3.14159265358979)
	enc.PutString("Hello, World!")

	// Decode
	dec := NewDecoder(enc.Bytes())

	u8, err := dec.ReadUint8()
	if err != nil {
		t.Fatalf("ReadUint8: %v", err)
	}
	if u8 != 0x42 {
		t.Errorf("uint8 = 0x%02X, want 0x42", u8)
	}

	u16, err := dec.ReadUint16()
	if err != nil {
		t.Fatalf("ReadUint16: %v", err)
	}
	if u16 != 9999 {
		t.Errorf("uint16 = %d, want 9999", u16)
	}

	u32, err := dec.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32: %v", err)
	}
	if u32 != 123456789 {
		t.Errorf("uint32 = %d, want 123456789", u32)
	}

	f64, err := dec.ReadFloat64()
	if err != nil {
		t.Fatalf("ReadFloat64: %v", err)
	}
	if f64 != 3.14159265358979 {
		t.Errorf("float64 = %v, want 3.14159265358979", f64)
	}

	str, err := dec.ReadString(13)
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if str != "Hello, World!" {
		t.Errorf("string = %q, want %q", str, "Hello, World!")
	}

	// Should have consumed every byte
	if dec.Remaining() != 0 {
		t.Errorf("Remaining() = %d, want 0 (all bytes consumed)", dec.Remaining())
	}
}

func TestRoundTrip_TLVField(t *testing.T) {
	// Simulate a full TLV encode/decode cycle for a MonetaryValue field.
	// This is the exact flow that happens when the C++ client sends a
	// deposit amount and the Go server decodes it.
	amount := 2500.75

	// Encode (what the C++ client does)
	enc := NewEncoder()
	enc.PutUint8(FieldMonetaryValue)    // tag
	enc.PutUint32(8)                    // length: float64 is 8 bytes
	enc.PutFloat64(amount)              // value

	// Decode (what our Go TLV decoder will do)
	dec := NewDecoder(enc.Bytes())

	tag, err := dec.ReadUint8()
	if err != nil {
		t.Fatalf("ReadUint8 (tag): %v", err)
	}
	if tag != FieldMonetaryValue {
		t.Errorf("tag = 0x%02X, want 0x%02X", tag, FieldMonetaryValue)
	}

	length, err := dec.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32 (length): %v", err)
	}
	if length != 8 {
		t.Errorf("length = %d, want 8", length)
	}

	value, err := dec.ReadFloat64()
	if err != nil {
		t.Fatalf("ReadFloat64 (value): %v", err)
	}
	if value != amount {
		t.Errorf("value = %v, want %v", value, amount)
	}

	if dec.Remaining() != 0 {
		t.Errorf("Remaining() = %d after reading full TLV field", dec.Remaining())
	}
}

func TestRoundTrip_MultipleFieldsAnyOrder(t *testing.T) {
	// The C++ client can send fields in any order. Verify we can encode
	// two fields and decode them back regardless of ordering.

	// Build: Currency field first, then AccountNumber
	enc := NewEncoder()

	// Currency TLV
	enc.PutUint8(FieldCurrency)
	enc.PutUint32(1) // length = 1
	enc.PutUint8(1)  // SGD = 1

	// AccountNumber TLV
	enc.PutUint8(FieldAccountNumber)
	enc.PutUint32(4) // length = 4
	enc.PutUint32(10042)

	// Decode and verify we get both fields regardless of order
	dec := NewDecoder(enc.Bytes())

	// First field: Currency
	tag1, _ := dec.ReadUint8()
	len1, _ := dec.ReadUint32()
	if tag1 != FieldCurrency || len1 != 1 {
		t.Errorf("field 1: tag=0x%02X len=%d, want tag=0x%02X len=1", tag1, len1, FieldCurrency)
	}
	curr, _ := dec.ReadUint8()
	if curr != 1 {
		t.Errorf("currency = %d, want 1 (SGD)", curr)
	}

	// Second field: AccountNumber
	tag2, _ := dec.ReadUint8()
	len2, _ := dec.ReadUint32()
	if tag2 != FieldAccountNumber || len2 != 4 {
		t.Errorf("field 2: tag=0x%02X len=%d, want tag=0x%02X len=4", tag2, len2, FieldAccountNumber)
	}
	accNo, _ := dec.ReadUint32()
	if accNo != 10042 {
		t.Errorf("account number = %d, want 10042", accNo)
	}

	if dec.Remaining() != 0 {
		t.Errorf("Remaining() = %d, want 0", dec.Remaining())
	}
}

// ─────────────────────────────────────────────────────────────────────
// Edge case: decoding from a buffer produced by the C++ client.
// These are manually constructed byte slices that match what the C++
// CommandEncoder::encode_message would produce.
// ─────────────────────────────────────────────────────────────────────

func TestDecodeCppOpenAccountRequest(t *testing.T) {
	// Simulate a C++ OpenAccount request payload (after the 5-byte semantics header).
	// Fields: Service=1, AccountOwnerName="Alice", AccountPassword="pass1234", Currency=1, MonetaryValue=500.0
	enc := NewEncoder()

	// Service (tag=0x01, len=1, val=1)
	enc.PutUint8(0x01)
	enc.PutUint32(1)
	enc.PutUint8(1) // OPEN

	// AccountOwnerName (tag=0x03, len=5, val="Alice")
	enc.PutUint8(0x03)
	enc.PutUint32(5)
	enc.PutString("Alice")

	// AccountPassword (tag=0x04, len=8, val="pass1234")
	enc.PutUint8(0x04)
	enc.PutUint32(8)
	enc.PutString("pass1234")

	// Currency (tag=0x08, len=1, val=1 SGD)
	enc.PutUint8(0x08)
	enc.PutUint32(1)
	enc.PutUint8(1)

	// MonetaryValue (tag=0x07, len=8, val=500.0)
	enc.PutUint8(0x07)
	enc.PutUint32(8)
	enc.PutFloat64(500.0)

	// Now decode the whole thing field by field
	dec := NewDecoder(enc.Bytes())
	fieldCount := 0

	for dec.Remaining() >= TLVHeaderSize {
		tag, err := dec.ReadUint8()
		if err != nil {
			t.Fatalf("field %d tag: %v", fieldCount, err)
		}

		length, err := dec.ReadUint32()
		if err != nil {
			t.Fatalf("field %d length: %v", fieldCount, err)
		}

		if !IsValidFieldID(tag) {
			t.Fatalf("field %d: unknown tag 0x%02X", fieldCount, tag)
		}

		// Just skip past the value — we're testing the loop structure, not the values
		if err := dec.Skip(int(length)); err != nil {
			t.Fatalf("field %d: skip %d bytes: %v", fieldCount, length, err)
		}

		fieldCount++
	}

	if fieldCount != 5 {
		t.Errorf("decoded %d fields, want 5", fieldCount)
	}

	if dec.Remaining() != 0 {
		t.Errorf("Remaining() = %d after decoding all fields", dec.Remaining())
	}
}

func TestDecodeCppMonitorRegistrationRequest(t *testing.T) {
	// Simulate a C++ Monitor registration request payload (after the 5-byte
	// semantics header). This uses the NEW fields added in the client rev2:
	//   - FieldMonitorUpdates (0x09): variable-length string
	//   - FieldMonitorTimeoutSeconds (0x0A): uint32 seconds
	enc := NewEncoder()

	// Service (tag=0x01, len=1, val=5 MONITOR)
	enc.PutUint8(FieldService)
	enc.PutUint32(1)
	enc.PutUint8(5) // ServiceMonitor

	// MonitorTimeoutSeconds (tag=0x0A, len=4, val=60)
	enc.PutUint8(FieldMonitorTimeoutSeconds)
	enc.PutUint32(4)
	enc.PutUint32(60) // 60 seconds

	// MonitorUpdates (tag=0x09, len=3, val="all")
	enc.PutUint8(FieldMonitorUpdates)
	enc.PutUint32(3)
	enc.PutString("all")

	// Decode and verify each field
	dec := NewDecoder(enc.Bytes())

	// Field 1: Service
	tag, _ := dec.ReadUint8()
	length, _ := dec.ReadUint32()
	if tag != FieldService || length != 1 {
		t.Fatalf("field 1: tag=0x%02X len=%d, want tag=0x%02X len=1", tag, length, FieldService)
	}
	svc, _ := dec.ReadUint8()
	if svc != 5 {
		t.Errorf("service = %d, want 5 (Monitor)", svc)
	}

	// Field 2: MonitorTimeoutSeconds
	tag, _ = dec.ReadUint8()
	length, _ = dec.ReadUint32()
	if tag != FieldMonitorTimeoutSeconds || length != 4 {
		t.Fatalf("field 2: tag=0x%02X len=%d, want tag=0x%02X len=4", tag, length, FieldMonitorTimeoutSeconds)
	}
	timeout, _ := dec.ReadUint32()
	if timeout != 60 {
		t.Errorf("timeout = %d, want 60", timeout)
	}

	// Field 3: MonitorUpdates
	tag, _ = dec.ReadUint8()
	length, _ = dec.ReadUint32()
	if tag != FieldMonitorUpdates || length != 3 {
		t.Fatalf("field 3: tag=0x%02X len=%d, want tag=0x%02X len=3", tag, length, FieldMonitorUpdates)
	}
	updates, _ := dec.ReadString(int(length))
	if updates != "all" {
		t.Errorf("monitor_updates = %q, want %q", updates, "all")
	}

	if dec.Remaining() != 0 {
		t.Errorf("Remaining() = %d after decoding all fields", dec.Remaining())
	}
}