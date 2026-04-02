package marshal

import (
	"encoding/binary"
	"net"
)

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
