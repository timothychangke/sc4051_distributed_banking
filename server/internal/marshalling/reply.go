package marshal

import (
	"net"
)


// ReplyHeaderSize is the minimum size of a reply packet with zero-length content.
const ReplyHeaderSize = 18

// MsgTypeReply is the message type byte for standard request/response replies.
// Duplicated here to avoid a circular import with the protocol package.
// Must stay in sync with protocol.MsgTypeReply (0x01).
const MsgTypeReply uint8 = 0x01

func BuildReply(requestID uint32, clientAddr *net.UDPAddr, statusCode uint8, content []byte) []byte {
	contentLen := 0
	if content != nil {
		contentLen = len(content)
	}

	// Pre-allocate the exact size so we don't waste time growing the slice
	enc := NewEncoderWithCap(ReplyHeaderSize + contentLen)

	// Byte 0: Message type: always 0x01 for a normal reply
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
	// Same deal: must be present for alignment even if unused.
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
