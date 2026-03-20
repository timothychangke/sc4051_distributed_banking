package semantics

import (
	"encoding/binary"
	"testing"
)

func TestParseHeader_Valid(t *testing.T) {
	// Build a valid 18-byte header:
	// Type=0, Flag=1, RID=42, IP=127.0.0.1, Port=9001, Status=0, Len=100
	data := make([]byte, HeaderSize+10)
	data[0] = 0
	data[1] = 1
	binary.BigEndian.PutUint32(data[2:6], 42)
	binary.BigEndian.PutUint32(data[6:10], 0x7F000001)
	binary.BigEndian.PutUint16(data[10:12], 9001)
	binary.BigEndian.PutUint16(data[12:14], 0)
	binary.BigEndian.PutUint32(data[14:18], 100)

	hdr, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if hdr.Type != 0 {
		t.Errorf("Expected Type=0, got %d", hdr.Type)
	}
	if hdr.Flag != 1 {
		t.Errorf("Expected Flag=1, got %d", hdr.Flag)
	}
	if hdr.RequestID != 42 {
		t.Errorf("Expected RequestID=42, got %d", hdr.RequestID)
	}
	if hdr.Port != 9001 {
		t.Errorf("Expected Port=9001, got %d", hdr.Port)
	}
	if hdr.ContentLen != 100 {
		t.Errorf("Expected ContentLen=100, got %d", hdr.ContentLen)
	}
}

func TestParseHeader_ExactMinimumSize(t *testing.T) {
	// A packet with exactly 18 bytes should still parse fine
	data := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(data[2:6], 99999)

	hdr, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("Unexpected error on exact-size packet: %v", err)
	}

	if hdr.RequestID != 99999 {
		t.Errorf("Expected RequestID=99999, got %d", hdr.RequestID)
	}
}

func TestParseHeader_TooShort(t *testing.T) {
	// Anything under 18 bytes is malformed
	data := make([]byte, HeaderSize-1)

	_, err := ParseHeader(data)
	if err != ErrPacketTooShort {
		t.Errorf("Expected ErrPacketTooShort, got %v", err)
	}
}

func TestParseHeader_BigEndianByteOrder(t *testing.T) {
	// Verify we're actually reading big-endian, not accidentally little-endian.
	// RequestID = 0x00000100 = 256 in big-endian
	data := make([]byte, HeaderSize)
	data[2] = 0x00
	data[3] = 0x00
	data[4] = 0x01
	data[5] = 0x00

	hdr, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if hdr.RequestID != 256 {
		t.Errorf("Expected RequestID=256 (big-endian), got %d — possible byte order bug", hdr.RequestID)
	}
}