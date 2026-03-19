package semantics

import (
	"encoding/binary"
	"testing"
)

func TestParseHeader_Valid(t *testing.T) {
	// Build a minimal valid packet: ServiceID=3 (Deposit), RequestID=42
	data := make([]byte, HeaderSize+10) // extra bytes simulate payload
	data[0] = 3
	binary.BigEndian.PutUint32(data[1:5], 42)

	hdr, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if hdr.ServiceID != 3 {
		t.Errorf("Expected ServiceID=3, got %d", hdr.ServiceID)
	}
	if hdr.RequestID != 42 {
		t.Errorf("Expected RequestID=42, got %d", hdr.RequestID)
	}
}

func TestParseHeader_ExactMinimumSize(t *testing.T) {
	// A packet with exactly 5 bytes should still parse fine
	data := make([]byte, HeaderSize)
	data[0] = 7
	binary.BigEndian.PutUint32(data[1:5], 99999)

	hdr, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("Unexpected error on exact-size packet: %v", err)
	}

	if hdr.ServiceID != 7 {
		t.Errorf("Expected ServiceID=7, got %d", hdr.ServiceID)
	}
	if hdr.RequestID != 99999 {
		t.Errorf("Expected RequestID=99999, got %d", hdr.RequestID)
	}
}

func TestParseHeader_TooShort(t *testing.T) {
	// Anything under 5 bytes is malformed
	shortPackets := [][]byte{
		{},
		{0x01},
		{0x01, 0x00},
		{0x01, 0x00, 0x00},
		{0x01, 0x00, 0x00, 0x00},
	}

	for i, data := range shortPackets {
		_, err := ParseHeader(data)
		if err != ErrPacketTooShort {
			t.Errorf("Case %d (len=%d): expected ErrPacketTooShort, got %v", i, len(data), err)
		}
	}
}

func TestParseHeader_BigEndianByteOrder(t *testing.T) {
	// Verify we're actually reading big-endian, not accidentally little-endian.
	// RequestID = 0x00000100 = 256 in big-endian
	data := []byte{0x01, 0x00, 0x00, 0x01, 0x00}

	hdr, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if hdr.RequestID != 256 {
		t.Errorf("Expected RequestID=256 (big-endian), got %d — possible byte order bug", hdr.RequestID)
	}
}