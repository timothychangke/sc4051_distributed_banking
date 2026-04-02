package marshal

import (
	"encoding/binary"
	"net"
)

// ─────────────────────────────────────────────────────────────────────
// Shared utility functions for the marshalling layer.
//
// These handle the quirky conversions between what the C++ client sends
// on the wire and what the Go banking service expects in its API.
// ─────────────────────────────────────────────────────────────────────

// IPv4ToUint32 converts a net.UDPAddr's IP into the big-endian uint32
// that the C++ MessageSerializer expects in the reply header.
//
// If the address isn't a valid IPv4 (e.g., it's IPv6 or nil), we return 0.
// The C++ client doesn't actually use this field for routing: it already
// knows its own address: but the bytes must be present in the reply or
// the fixed-offset deserializer reads garbage for everything after it.
func IPv4ToUint32(addr *net.UDPAddr) uint32 {
	if addr == nil || addr.IP == nil {
		return 0
	}

	// net.IP can be 4 bytes (IPv4) or 16 bytes (IPv4-in-IPv6 or IPv6).
	// To4() normalises both forms into 4 bytes, returning nil if it's
	// genuinely an IPv6 address that can't be represented as IPv4.
	ip4 := addr.IP.To4()
	if ip4 == nil {
		return 0
	}

	// The 4 bytes of an IPv4 address are already in network byte order
	// (most significant octet first), which is the same as big-endian.
	// So we can just read them directly as a uint32.
	return binary.BigEndian.Uint32(ip4)
}

// PasswordStringToFixed converts a variable-length password string from
// the wire into the [8]byte array that the Go banking service expects.
//
// The C++ client sends passwords as variable-length strings: could be
// 3 bytes, could be 8, could theoretically be longer. The Go banking
// layer expects exactly [8]byte. We handle the mismatch here:
//   - Short passwords get zero-padded on the right
//   - Passwords longer than 8 bytes get truncated (shouldn't happen in
//     practice since the client UI enforces the limit, but defense in depth)
func PasswordStringToFixed(pw string) [8]byte {
	var fixed [8]byte

	// copy() handles the length mismatch gracefully:
	// it copies min(len(src), len(dst)) bytes. If pw is shorter than 8,
	// the remaining bytes stay as zeroes. If longer, only the first 8
	// bytes are copied.
	copy(fixed[:], pw)

	return fixed
}

// Uint32ToIPv4 does the reverse of IPv4ToUint32: turns a big-endian
// uint32 back into a net.IP. Useful if we ever need to reconstruct a
// client address from the wire format (e.g., in test code).
func Uint32ToIPv4(v uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, v)
	return ip
}
