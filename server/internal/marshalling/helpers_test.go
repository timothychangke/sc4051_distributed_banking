package marshal

import (
	"net"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────
// Helper function tests.
// ─────────────────────────────────────────────────────────────────────

// --- IPv4ToUint32 tests ---

func TestIPv4ToUint32_StandardAddress(t *testing.T) {
	addr := &net.UDPAddr{IP: net.IPv4(192, 168, 1, 100), Port: 8080}
	got := IPv4ToUint32(addr)

	// 192.168.1.100 = (192 << 24) | (168 << 16) | (1 << 8) | 100
	expected := uint32(192)<<24 | uint32(168)<<16 | uint32(1)<<8 | uint32(100)
	if got != expected {
		t.Errorf("want 0x%08X, got 0x%08X", expected, got)
	}
}

func TestIPv4ToUint32_Loopback(t *testing.T) {
	addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 2222}
	got := IPv4ToUint32(addr)

	expected := uint32(127)<<24 | uint32(0)<<16 | uint32(0)<<8 | uint32(1)
	if got != expected {
		t.Errorf("want 0x%08X, got 0x%08X", expected, got)
	}
}

func TestIPv4ToUint32_AllZeros(t *testing.T) {
	addr := &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 0}
	got := IPv4ToUint32(addr)
	if got != 0 {
		t.Errorf("want 0, got 0x%08X", got)
	}
}

func TestIPv4ToUint32_AllOnes(t *testing.T) {
	addr := &net.UDPAddr{IP: net.IPv4(255, 255, 255, 255), Port: 0}
	got := IPv4ToUint32(addr)
	if got != 0xFFFFFFFF {
		t.Errorf("want 0xFFFFFFFF, got 0x%08X", got)
	}
}

func TestIPv4ToUint32_NilAddr(t *testing.T) {
	got := IPv4ToUint32(nil)
	if got != 0 {
		t.Errorf("nil addr: want 0, got 0x%08X", got)
	}
}

func TestIPv4ToUint32_NilIP(t *testing.T) {
	addr := &net.UDPAddr{IP: nil, Port: 8080}
	got := IPv4ToUint32(addr)
	if got != 0 {
		t.Errorf("nil IP: want 0, got 0x%08X", got)
	}
}

func TestIPv4ToUint32_IPv6Fallback(t *testing.T) {
	// Pure IPv6 that can't be represented as IPv4 should return 0
	addr := &net.UDPAddr{IP: net.ParseIP("::1"), Port: 8080}
	got := IPv4ToUint32(addr)
	if got != 0 {
		t.Errorf("IPv6 loopback: want 0, got 0x%08X", got)
	}
}

func TestIPv4ToUint32_IPv4MappedIPv6(t *testing.T) {
	// IPv4-mapped IPv6 addresses (like ::ffff:192.168.1.1) should work
	// because To4() handles the conversion.
	addr := &net.UDPAddr{
		IP:   net.ParseIP("::ffff:192.168.1.1"),
		Port: 8080,
	}
	got := IPv4ToUint32(addr)
	expected := uint32(192)<<24 | uint32(168)<<16 | uint32(1)<<8 | uint32(1)
	if got != expected {
		t.Errorf("IPv4-mapped IPv6: want 0x%08X, got 0x%08X", expected, got)
	}
}

// --- Uint32ToIPv4 tests (reverse of IPv4ToUint32) ---

func TestUint32ToIPv4_RoundTrip(t *testing.T) {
	original := &net.UDPAddr{IP: net.IPv4(10, 0, 0, 42), Port: 9999}
	asUint32 := IPv4ToUint32(original)
	backToIP := Uint32ToIPv4(asUint32)

	// Compare the 4-byte form
	expected := original.IP.To4()
	if !backToIP.Equal(expected) {
		t.Errorf("round-trip failed: started with %s, got %s", expected, backToIP)
	}
}

func TestUint32ToIPv4_Zero(t *testing.T) {
	ip := Uint32ToIPv4(0)
	expected := net.IPv4(0, 0, 0, 0).To4()
	if !ip.Equal(expected) {
		t.Errorf("want %s, got %s", expected, ip)
	}
}

// --- PasswordStringToFixed tests ---

func TestPasswordStringToFixed_ExactLength(t *testing.T) {
	pw := "12345678" // exactly 8 bytes
	got := PasswordStringToFixed(pw)
	expected := [8]byte{'1', '2', '3', '4', '5', '6', '7', '8'}
	if got != expected {
		t.Errorf("want %v, got %v", expected, got)
	}
}

func TestPasswordStringToFixed_ShortPassword(t *testing.T) {
	pw := "abc" // 3 bytes: should be zero-padded
	got := PasswordStringToFixed(pw)
	expected := [8]byte{'a', 'b', 'c', 0, 0, 0, 0, 0}
	if got != expected {
		t.Errorf("want %v, got %v", expected, got)
	}
}

func TestPasswordStringToFixed_EmptyPassword(t *testing.T) {
	got := PasswordStringToFixed("")
	expected := [8]byte{}
	if got != expected {
		t.Errorf("want %v, got %v", expected, got)
	}
}

func TestPasswordStringToFixed_LongPassword(t *testing.T) {
	pw := "verylongpassword" // 16 bytes: truncated to first 8
	got := PasswordStringToFixed(pw)
	expected := [8]byte{'v', 'e', 'r', 'y', 'l', 'o', 'n', 'g'}
	if got != expected {
		t.Errorf("want %v, got %v", expected, got)
	}
}

func TestPasswordStringToFixed_SingleChar(t *testing.T) {
	got := PasswordStringToFixed("X")
	expected := [8]byte{'X', 0, 0, 0, 0, 0, 0, 0}
	if got != expected {
		t.Errorf("want %v, got %v", expected, got)
	}
}

func TestPasswordStringToFixed_SpecialChars(t *testing.T) {
	// Make sure binary chars don't get mangled
	pw := "p@ss\x00w0r"
	got := PasswordStringToFixed(pw)
	expected := [8]byte{'p', '@', 's', 's', 0x00, 'w', '0', 'r'}
	if got != expected {
		t.Errorf("want %v, got %v", expected, got)
	}
}
