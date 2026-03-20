package marshal

import (
	"encoding/binary"
	"net"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────
// Reply builder tests.
//
// These verify byte-for-byte that BuildReply produces packets the C++
// MessageSerializer::deserialize() can parse. The most critical checks
// are that the header is exactly 17 bytes and that every field sits at
// the offset the C++ code expects.
// ─────────────────────────────────────────────────────────────────────

// testAddr returns a standard test UDP address for deterministic output.
func testAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   net.IPv4(192, 168, 1, 100),
		Port: 12345,
	}
}

func TestBuildReply_EmptyContent_ExactSize(t *testing.T) {
	reply := BuildReply(42, testAddr(), 0, nil)

	// The C++ client expects exactly 18 bytes for an empty reply:
	// 1 (MsgType) + 1 (Flag) + 4 (RID) + 4 (IP) + 2 (Port) + 2 (Status) + 4 (Len)
	if len(reply) != ReplyHeaderSize {
		t.Fatalf("empty reply should be %d bytes, got %d", ReplyHeaderSize, len(reply))
	}
}

func TestBuildReply_MsgTypeAndFlag(t *testing.T) {
	reply := BuildReply(1, testAddr(), 0, nil)

	// Byte 0: MsgType must be 0x01 (MsgTypeReply)
	if reply[0] != MsgTypeReply {
		t.Errorf("byte 0 (MsgType): want 0x%02X, got 0x%02X", MsgTypeReply, reply[0])
	}
	// Byte 1: Flag must be 0x00 for normal replies
	if reply[1] != 0x00 {
		t.Errorf("byte 1 (Flag): want 0x00, got 0x%02X", reply[1])
	}
}

func TestBuildReply_RequestID(t *testing.T) {
	reply := BuildReply(0xDEADBEEF, testAddr(), 0, nil)

	// Bytes 2-5: RequestID in big-endian
	gotID := binary.BigEndian.Uint32(reply[2:6])
	if gotID != 0xDEADBEEF {
		t.Errorf("bytes 2-5 (RequestID): want 0xDEADBEEF, got 0x%08X", gotID)
	}
}

func TestBuildReply_IPv4Field(t *testing.T) {
	addr := testAddr() // 192.168.1.100
	reply := BuildReply(1, addr, 0, nil)

	// Bytes 6-9: IPv4 as big-endian uint32
	gotIP := binary.BigEndian.Uint32(reply[6:10])
	// 192.168.1.100 = 0xC0A80164
	expectedIP := uint32(192)<<24 | uint32(168)<<16 | uint32(1)<<8 | uint32(100)
	if gotIP != expectedIP {
		t.Errorf("bytes 6-9 (IPv4): want 0x%08X, got 0x%08X", expectedIP, gotIP)
	}
}

func TestBuildReply_PortField(t *testing.T) {
	addr := testAddr() // port 12345
	reply := BuildReply(1, addr, 0, nil)

	// Bytes 10-11: Port as big-endian uint16
	gotPort := binary.BigEndian.Uint16(reply[10:12])
	if gotPort != 12345 {
		t.Errorf("bytes 10-11 (Port): want 12345, got %d", gotPort)
	}
}

func TestBuildReply_StatusCodeIsUint16(t *testing.T) {
	// StatusCode MUST occupy 2 bytes, not 1.
	reply := BuildReply(1, testAddr(), 5, nil) // status 5 = insufficient funds

	// Bytes 12-13: StatusCode as big-endian uint16
	gotStatus := binary.BigEndian.Uint16(reply[12:14])
	if gotStatus != 5 {
		t.Errorf("bytes 12-13 (StatusCode): want 5, got %d", gotStatus)
	}

	// Also verify the high byte is 0 (it's a small number)
	if reply[12] != 0x00 {
		t.Errorf("StatusCode high byte should be 0x00, got 0x%02X", reply[12])
	}
	if reply[13] != 0x05 {
		t.Errorf("StatusCode low byte should be 0x05, got 0x%02X", reply[13])
	}
}

func TestBuildReply_ContentLength_Empty(t *testing.T) {
	reply := BuildReply(1, testAddr(), 0, nil)

	// Bytes 14-17: Content length = 0
	gotLen := binary.BigEndian.Uint32(reply[14:18])
	if gotLen != 0 {
		t.Errorf("bytes 14-17 (ContentLen): want 0, got %d", gotLen)
	}
}

func TestBuildReply_ContentLength_WithBody(t *testing.T) {
	content := []byte("hello world")
	reply := BuildReply(1, testAddr(), 0, content)

	// Bytes 14-17: Content length
	gotLen := binary.BigEndian.Uint32(reply[14:18])
	if gotLen != uint32(len(content)) {
		t.Errorf("bytes 14-17 (ContentLen): want %d, got %d", len(content), gotLen)
	}

	// Total size should be header + content
	expectedTotal := ReplyHeaderSize + len(content)
	if len(reply) != expectedTotal {
		t.Errorf("total reply size: want %d, got %d", expectedTotal, len(reply))
	}

	// Bytes 18+: The content itself
	gotContent := reply[18:]
	if string(gotContent) != "hello world" {
		t.Errorf("content body: want 'hello world', got '%s'", string(gotContent))
	}
}

func TestBuildReply_WithTLVContent(t *testing.T) {
	// Simulate what a Deposit reply looks like: TLV-encoded new balance
	content := EncodeTLVFields([]TLVField{
		TLVFloat64(FieldMonetaryValue, 1500.75),
	})
	reply := BuildReply(99, testAddr(), 0, content)

	// Verify the content length in the header matches
	headerContentLen := binary.BigEndian.Uint32(reply[14:18])
	if headerContentLen != uint32(len(content)) {
		t.Errorf("content length in header: want %d, got %d", len(content), headerContentLen)
	}

	// Decode the content portion back as TLV to verify it round-trips
	contentBytes := reply[18:]
	cmd, err := DecodeTLV(contentBytes)
	if err != nil {
		t.Fatalf("failed to decode TLV content from reply: %v", err)
	}
	if cmd.MonetaryValue == nil || *cmd.MonetaryValue != 1500.75 {
		t.Errorf("round-trip balance: want 1500.75, got %v", cmd.MonetaryValue)
	}
}

func TestBuildReply_EmptyContent_NilVsEmptySlice(t *testing.T) {
	// Both nil and empty slice should produce the same 18-byte reply
	replyNil := BuildReply(1, testAddr(), 0, nil)
	replyEmpty := BuildReply(1, testAddr(), 0, []byte{})

	if len(replyNil) != ReplyHeaderSize {
		t.Errorf("nil content: want %d bytes, got %d", ReplyHeaderSize, len(replyNil))
	}
	if len(replyEmpty) != ReplyHeaderSize {
		t.Errorf("empty content: want %d bytes, got %d", ReplyHeaderSize, len(replyEmpty))
	}
}

func TestBuildReply_IPv6AddrFallback(t *testing.T) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("::1"),
		Port: 8080,
	}
	reply := BuildReply(1, addr, 0, nil)

	if len(reply) != ReplyHeaderSize {
		t.Fatalf("IPv6 reply should be %d bytes, got %d", ReplyHeaderSize, len(reply))
	}

	// IPv4 field covers bytes 6-9
	gotIP := binary.BigEndian.Uint32(reply[6:10])
	if gotIP != 0 {
		t.Errorf("IPv4 for IPv6 addr: want 0, got 0x%08X", gotIP)
	}
}

func TestBuildReply_AllStatusCodes(t *testing.T) {
	// Verify that all our status codes encode correctly as uint16 at offset 12
	codes := []uint8{0, 1, 2, 3, 4, 5, 6, 7}

	for _, code := range codes {
		reply := BuildReply(1, testAddr(), code, nil)
		gotStatus := binary.BigEndian.Uint16(reply[12:14])
		if gotStatus != uint16(code) {
			t.Errorf("status code %d: wire value is %d", code, gotStatus)
		}
	}
}
