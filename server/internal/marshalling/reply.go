package marshal

import (
	"net"
)

// ─────────────────────────────────────────────────────────────────────
// Reply packet builder.
//
// The C++ client's MessageSerializer::deserialize() expects this EXACT
// byte layout for every reply. If any field is the wrong size or at the
// wrong offset, the client reads garbage for all subsequent fields.
//
//
// ┌──────────┬──────────┬──────────────┬──────────┬──────────┬────────────────┬─────────────────┐
// │ MsgType  │ Flag     │ RequestID    │ IPv4     │ Port     │ StatusCode     │ Content          │
// │ (1 byte) │ (1 byte) │ (4 bytes BE) │ (4B BE)  │ (2B BE)  │ (2 bytes BE)   │ [4B len][N data] │
// │ 0x01     │ 0x00     │              │          │          │                │                  │
// └──────────┴──────────┴──────────────┴──────────┴──────────┴────────────────┴─────────────────┘
//
// Total fixed header = 1 + 1 + 4 + 4 + 2 + 2 + 4 = 18 bytes minimum (with empty content).
//
// GOTCHA #1: StatusCode is 2 bytes (uint16) on the wire, even though our
// protocol.go defines them as uint8. The C++ deserializer reads with ntohs().
// If you write only 1 byte here, everything after it shifts and the client
// reads garbage for content_len.
//
// GOTCHA #2: The IPv4 and Port fields MUST be present. The main.go comments
// show a simpler layout without them, but the C++ code is the source of truth
// and it reads exactly 6 bytes between RequestID and StatusCode.
// ─────────────────────────────────────────────────────────────────────

// ReplyHeaderSize is the minimum size of a reply packet with zero-length content.
const ReplyHeaderSize = 18

// MsgTypeReply is the message type byte for standard request/response replies.
// Duplicated here to avoid a circular import with the protocol package.
// Must stay in sync with protocol.MsgTypeReply (0x01).
const MsgTypeReply uint8 = 0x01

// BuildReply constructs a complete reply packet that the C++ MessageSerializer
// can deserialize. The requestID is echoed back so the client can match this
// reply to its outstanding request. The clientAddr is included because the C++
// wire format expects it — we echo the client's own address back to them.
//
// The content parameter is the service-specific response body (e.g., TLV-encoded
// new balance after a deposit). Pass nil for replies that carry no content
// (e.g., CloseAccount success, Monitor ack).
func BuildReply(requestID uint32, clientAddr *net.UDPAddr, statusCode uint8, content []byte) []byte {
	contentLen := 0
	if content != nil {
		contentLen = len(content)
	}

	// Pre-allocate the exact size so we don't waste time growing the slice
	enc := NewEncoderWithCap(ReplyHeaderSize + contentLen)

	// Byte 0: Message type — always 0x01 for a normal reply
	enc.PutUint8(MsgTypeReply)

	// Byte 1: Invocation flag. The C++ client expects an 18-byte header including
	// this flag byte. For replies, we use 0 (normal).
	enc.PutUint8(0)

	// Bytes 2-5: Echo back the request ID so the client can match it
	enc.PutUint32(requestID)

	// Bytes 6-9: Client's IPv4 address as a big-endian uint32.
	// The C++ side doesn't really use this for routing (it already knows
	// its own address), but the bytes MUST be here or the fixed-offset
	// deserialization breaks. We echo back the client's IP.
	enc.PutUint32(IPv4ToUint32(clientAddr))

	// Bytes 10-11: Client's port as a big-endian uint16.
	// Same deal — must be present for alignment even if unused.
	enc.PutUint16(uint16(clientAddr.Port))

	// Bytes 12-13: Status code widened to uint16.
	// Our protocol.go uses uint8, but the C++ MessageSerializer reads 2 bytes
	// with ntohs(). Writing only 1 byte here is the #1 most common bug.
	enc.PutUint16(uint16(statusCode))

	// Bytes 14-17: Content length as big-endian uint32
	enc.PutUint32(uint32(contentLen))

	// Bytes 18+: The actual response body (if any)
	if contentLen > 0 {
		enc.PutBytes(content)
	}

	return enc.Bytes()
}

// BuildErrorReply is a shorthand for replies that carry no content body.
// Most error responses only need the status code to tell the client what went wrong.
func BuildErrorReply(requestID uint32, clientAddr *net.UDPAddr, statusCode uint8) []byte {
	return BuildReply(requestID, clientAddr, statusCode, nil)
}