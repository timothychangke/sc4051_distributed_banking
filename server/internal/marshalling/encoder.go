package marshal

import (
	"encoding/binary"
	"math"
)


// NewEncoderWithCap pre-allocates the internal buffer to avoid repeated
// slice growth when you know roughly how big the output will be.
func NewEncoderWithCap(capacity int) *Encoder {
	return &Encoder{
		buf: make([]byte, 0, capacity),
	}
}

// PutUint8 appends a single byte. No endian conversion needed.
func (e *Encoder) PutUint8(v uint8) {
	e.buf = append(e.buf, v)
}

// PutUint16 appends a 16-bit unsigned integer in big-endian order.
// The C++ client uses htons() for port numbers and status codes.
func (e *Encoder) PutUint16(v uint16) {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	e.buf = append(e.buf, b...)
}

// PutUint32 appends a 32-bit unsigned integer in big-endian order.
// The C++ client uses htonl() for account numbers, request IDs,
// IPv4 addresses, and TLV field lengths.
func (e *Encoder) PutUint32(v uint32) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	e.buf = append(e.buf, b...)
}

// PutFloat64 appends a 64-bit IEEE 754 double in big-endian order.
//
// The wire bytes are identical. No special handling required.
func (e *Encoder) PutFloat64(v float64) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(v))
	e.buf = append(e.buf, b...)
}

// PutString appends raw string bytes without any length prefix.
// The caller is responsible for writing the length separately (e.g. as
// part of a TLV header or a length-prefixed string field).
func (e *Encoder) PutString(s string) {
	e.buf = append(e.buf, []byte(s)...)
}

// PutBytes appends a raw byte slice as-is. Useful for pre-encoded
// content blobs or copying data from one buffer to another.
func (e *Encoder) PutBytes(b []byte) {
	e.buf = append(e.buf, b...)
}

// PutLengthPrefixedString writes a [4-byte BE length][N-byte data] pair.
// This is the standard string encoding used in the TLV payload and the
// callback packet's HolderName field.
func (e *Encoder) PutLengthPrefixedString(s string) {
	e.PutUint32(uint32(len(s)))
	e.PutString(s)
}

func (e *Encoder) Bytes() []byte {
	return e.buf
}

// Len returns how many bytes have been written so far.
func (e *Encoder) Len() int {
	return len(e.buf)
}

// Reset clears the buffer so the encoder can be reused without
// allocating a new one. Keeps the underlying capacity.
func (e *Encoder) Reset() {
	e.buf = e.buf[:0]
}
