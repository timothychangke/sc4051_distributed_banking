package marshal

import (
	"encoding/binary"
	"math"
	"testing"
)


func TestPutUint8(t *testing.T) {
	enc := NewEncoder()
	enc.PutUint8(0x00)
	enc.PutUint8(0xFF)
	enc.PutUint8(0x42)

	got := enc.Bytes()
	want := []byte{0x00, 0xFF, 0x42}
	assertBytesEqual(t, got, want)
}

func TestPutUint16(t *testing.T) {
	enc := NewEncoder()

	
	enc.PutUint16(0)
	
	enc.PutUint16(1234)
	
	enc.PutUint16(0xFFFF)

	got := enc.Bytes()
	want := []byte{
		0x00, 0x00, 
		0x04, 0xD2, 
		0xFF, 0xFF, 
	}
	assertBytesEqual(t, got, want)
}

func TestPutUint32(t *testing.T) {
	enc := NewEncoder()

	
	enc.PutUint32(0)
	
	enc.PutUint32(10000)
	
	enc.PutUint32(0xFFFFFFFF)

	got := enc.Bytes()
	want := []byte{
		0x00, 0x00, 0x00, 0x00, 
		0x00, 0x00, 0x27, 0x10, 
		0xFF, 0xFF, 0xFF, 0xFF, 
	}
	assertBytesEqual(t, got, want)
}

func TestPutFloat64(t *testing.T) {
	tests := []struct {
		name string
		val  float64
	}{
		{"zero", 0.0},
		{"positive_integer", 100.0},
		{"typical_balance", 1234.56},
		{"negative", -500.25},
		{"very_small", 0.01},
		{"large", 999999999.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := NewEncoder()
			enc.PutFloat64(tt.val)

			got := enc.Bytes()
			if len(got) != 8 {
				t.Fatalf("PutFloat64 wrote %d bytes, want 8", len(got))
			}

			
			bits := binary.BigEndian.Uint64(got)
			roundTrip := math.Float64frombits(bits)
			if roundTrip != tt.val {
				t.Errorf("round-trip failed: put %v, got back %v", tt.val, roundTrip)
			}
		})
	}
}

func TestPutFloat64_WireBytes(t *testing.T) {
	
	
	enc := NewEncoder()
	enc.PutFloat64(100.50)

	got := enc.Bytes()

	
	
	
	want := []byte{0x40, 0x59, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00}
	assertBytesEqual(t, got, want)
}

func TestPutString(t *testing.T) {
	enc := NewEncoder()
	enc.PutString("Alice")

	got := enc.Bytes()
	want := []byte{'A', 'l', 'i', 'c', 'e'}
	assertBytesEqual(t, got, want)
}

func TestPutString_Empty(t *testing.T) {
	enc := NewEncoder()
	enc.PutString("")

	if enc.Len() != 0 {
		t.Errorf("empty PutString wrote %d bytes, want 0", enc.Len())
	}
}

func TestPutBytes(t *testing.T) {
	enc := NewEncoder()
	enc.PutBytes([]byte{0xDE, 0xAD, 0xBE, 0xEF})

	got := enc.Bytes()
	want := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	assertBytesEqual(t, got, want)
}

func TestPutBytes_Nil(t *testing.T) {
	enc := NewEncoder()
	enc.PutBytes(nil)

	if enc.Len() != 0 {
		t.Errorf("nil PutBytes wrote %d bytes, want 0", enc.Len())
	}
}

func TestPutLengthPrefixedString(t *testing.T) {
	enc := NewEncoder()
	enc.PutLengthPrefixedString("Bob")

	got := enc.Bytes()
	
	want := []byte{
		0x00, 0x00, 0x00, 0x03, 
		'B', 'o', 'b', 
	}
	assertBytesEqual(t, got, want)
}

func TestPutLengthPrefixedString_Empty(t *testing.T) {
	enc := NewEncoder()
	enc.PutLengthPrefixedString("")

	got := enc.Bytes()
	
	want := []byte{0x00, 0x00, 0x00, 0x00}
	assertBytesEqual(t, got, want)
}





func TestLen(t *testing.T) {
	enc := NewEncoder()
	if enc.Len() != 0 {
		t.Errorf("new encoder Len() = %d, want 0", enc.Len())
	}

	enc.PutUint8(1)
	if enc.Len() != 1 {
		t.Errorf("after PutUint8, Len() = %d, want 1", enc.Len())
	}

	enc.PutUint32(42)
	if enc.Len() != 5 {
		t.Errorf("after PutUint32, Len() = %d, want 5", enc.Len())
	}
}

func TestReset(t *testing.T) {
	enc := NewEncoder()
	enc.PutUint32(12345)
	enc.PutString("hello")

	enc.Reset()

	if enc.Len() != 0 {
		t.Errorf("after Reset, Len() = %d, want 0", enc.Len())
	}
	if len(enc.Bytes()) != 0 {
		t.Errorf("after Reset, Bytes() has %d bytes, want 0", len(enc.Bytes()))
	}
}

func TestNewEncoderWithCap(t *testing.T) {
	enc := NewEncoderWithCap(256)
	if enc.Len() != 0 {
		t.Errorf("NewEncoderWithCap Len() = %d, want 0", enc.Len())
	}

	
	enc.PutUint8(0x42)
	if enc.Bytes()[0] != 0x42 {
		t.Errorf("first byte = 0x%02X, want 0x42", enc.Bytes()[0])
	}
}






func TestEncoderBuildsValidTLVField(t *testing.T) {
	
	
	
	
	enc := NewEncoder()
	enc.PutUint8(FieldAccountNumber) 
	enc.PutUint32(4)                 
	enc.PutUint32(10000)             

	got := enc.Bytes()
	want := []byte{
		0x02,                   
		0x00, 0x00, 0x00, 0x04, 
		0x00, 0x00, 0x27, 0x10, 
	}
	assertBytesEqual(t, got, want)
}

func TestEncoderBuildsValidTLVStringField(t *testing.T) {
	
	
	
	
	enc := NewEncoder()
	name := "Alice"
	enc.PutUint8(FieldAccountOwnerName)
	enc.PutUint32(uint32(len(name)))
	enc.PutString(name)

	got := enc.Bytes()
	want := []byte{
		0x03,                   
		0x00, 0x00, 0x00, 0x05, 
		'A', 'l', 'i', 'c', 'e', 
	}
	assertBytesEqual(t, got, want)
}

func TestEncoderBuildsReplyHeader(t *testing.T) {
	
	
	
	
	enc := NewEncoder()
	enc.PutUint8(0x01)        
	enc.PutUint32(42)         
	enc.PutUint32(0xC0A80001) 
	enc.PutUint16(12345)      
	enc.PutUint16(0)          
	enc.PutUint32(0)          

	got := enc.Bytes()
	if len(got) != 17 {
		t.Fatalf("reply header = %d bytes, want exactly 17", len(got))
	}

	
	if got[0] != 0x01 {
		t.Errorf("MsgType = 0x%02X, want 0x01", got[0])
	}

	
	reqID := binary.BigEndian.Uint32(got[1:5])
	if reqID != 42 {
		t.Errorf("RequestID = %d, want 42", reqID)
	}

	
	status := binary.BigEndian.Uint16(got[11:13])
	if status != 0 {
		t.Errorf("StatusCode = %d, want 0", status)
	}
}

func TestEncoderSequentialWrites(t *testing.T) {
	
	enc := NewEncoder()
	enc.PutUint8(0xAA)
	enc.PutUint16(0xBBCC)
	enc.PutUint32(0xDDEEFF00)
	enc.PutFloat64(1.0)
	enc.PutString("hi")

	
	if enc.Len() != 17 {
		t.Errorf("total Len() = %d, want 17", enc.Len())
	}

	got := enc.Bytes()
	
	if got[0] != 0xAA {
		t.Errorf("first byte = 0x%02X, want 0xAA", got[0])
	}
	
	if got[15] != 'h' || got[16] != 'i' {
		t.Errorf("last two bytes = [0x%02X 0x%02X], want ['h' 'i']", got[15], got[16])
	}
}





func assertBytesEqual(t *testing.T, got, want []byte) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d bytes, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("byte mismatch at index %d: got 0x%02X, want 0x%02X\n  got:  %v\n  want: %v", i, got[i], want[i], got, want)
		}
	}
}
