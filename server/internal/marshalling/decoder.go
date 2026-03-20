package marshal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// ─────────────────────────────────────────────────────────────────────
// Decoder reads primitives from a byte slice at a moving offset.
//
// Every Read* method checks bounds before touching memory, advances
// the offset on success, and returns a clear error on buffer underflow.
// This mirrors the C++ CommandEncoder::decode_* functions which do the
// same bounds check before each memcpy.
//
// Usage:
//   dec := NewDecoder(wireBytes)
//   msgType, err := dec.ReadUint8()
//   reqID, err := dec.ReadUint32()
//   balance, err := dec.ReadFloat64()
// ─────────────────────────────────────────────────────────────────────

var (
	// ErrBufferUnderflow means we tried to read past the end of the buffer.
	// This usually indicates a truncated packet or a wrong field length.
	ErrBufferUnderflow = errors.New("read past end of buffer")
)

// ReadUint8 reads a single byte and advances the offset by 1.
func (d *Decoder) ReadUint8() (uint8, error) {
	if d.offset+1 > len(d.buf) {
		return 0, fmt.Errorf("ReadUint8 at offset %d: %w", d.offset, ErrBufferUnderflow)
	}

	v := d.buf[d.offset]
	d.offset++
	return v, nil
}

// ReadUint16 reads a big-endian 16-bit unsigned integer.
// The C++ client uses ntohs() when decoding these (ports, status codes).
func (d *Decoder) ReadUint16() (uint16, error) {
	const size = 2
	if d.offset+size > len(d.buf) {
		return 0, fmt.Errorf("ReadUint16 at offset %d: %w", d.offset, ErrBufferUnderflow)
	}

	v := binary.BigEndian.Uint16(d.buf[d.offset : d.offset+size])
	d.offset += size
	return v, nil
}

// ReadUint32 reads a big-endian 32-bit unsigned integer.
// The C++ client uses ntohl() when decoding these (account numbers,
// request IDs, TLV field lengths, IPv4 addresses).
func (d *Decoder) ReadUint32() (uint32, error) {
	const size = 4
	if d.offset+size > len(d.buf) {
		return 0, fmt.Errorf("ReadUint32 at offset %d: %w", d.offset, ErrBufferUnderflow)
	}

	v := binary.BigEndian.Uint32(d.buf[d.offset : d.offset+size])
	d.offset += size
	return v, nil
}

// ReadFloat64 reads a big-endian IEEE 754 64-bit double.
//
// On the wire this is 8 bytes that the C++ side produced by:
//   1. memcpy(uint64, double)   — reinterpret bits
//   2. manual htonll            — swap to big-endian
//
// We reverse it with:
//   1. binary.BigEndian.Uint64  — read as big-endian uint64
//   2. math.Float64frombits     — reinterpret back to float64
func (d *Decoder) ReadFloat64() (float64, error) {
	const size = 8
	if d.offset+size > len(d.buf) {
		return 0, fmt.Errorf("ReadFloat64 at offset %d: %w", d.offset, ErrBufferUnderflow)
	}

	bits := binary.BigEndian.Uint64(d.buf[d.offset : d.offset+size])
	d.offset += size
	return math.Float64frombits(bits), nil
}

// ReadString reads exactly `length` bytes and returns them as a string.
// No null-terminator handling — the C++ side sends raw bytes without one.
func (d *Decoder) ReadString(length int) (string, error) {
	if length < 0 {
		return "", fmt.Errorf("ReadString: negative length %d", length)
	}
	if d.offset+length > len(d.buf) {
		return "", fmt.Errorf("ReadString(%d) at offset %d: %w", length, d.offset, ErrBufferUnderflow)
	}

	s := string(d.buf[d.offset : d.offset+length])
	d.offset += length
	return s, nil
}

// ReadBytes reads exactly `length` bytes and returns a copy.
// Returns a new slice so the caller can't accidentally mutate the
// decoder's underlying buffer.
func (d *Decoder) ReadBytes(length int) ([]byte, error) {
	if length < 0 {
		return nil, fmt.Errorf("ReadBytes: negative length %d", length)
	}
	if d.offset+length > len(d.buf) {
		return nil, fmt.Errorf("ReadBytes(%d) at offset %d: %w", length, d.offset, ErrBufferUnderflow)
	}

	out := make([]byte, length)
	copy(out, d.buf[d.offset:d.offset+length])
	d.offset += length
	return out, nil
}

// Remaining returns how many unread bytes are left in the buffer.
func (d *Decoder) Remaining() int {
	return len(d.buf) - d.offset
}

// Offset returns the current read position. Useful for debugging
// when a decode goes wrong and you need to know where it derailed.
func (d *Decoder) Offset() int {
	return d.offset
}

// Skip advances the offset by n bytes without reading them.
// Handy for jumping past a header you've already parsed elsewhere
// (e.g. the 5-byte semantics header before the TLV payload).
func (d *Decoder) Skip(n int) error {
	if n < 0 {
		return fmt.Errorf("Skip: negative count %d", n)
	}
	if d.offset+n > len(d.buf) {
		return fmt.Errorf("Skip(%d) at offset %d: %w", n, d.offset, ErrBufferUnderflow)
	}

	d.offset += n
	return nil
}