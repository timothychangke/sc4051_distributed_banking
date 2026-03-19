package semantics

import (
	"encoding/binary"
	"errors"
)

// Wire format contract to NOTE FOR ZHIXUAN:
//
//   Byte 0       : ServiceID  (uint8)
//   Bytes 1–4    : RequestID  (uint32, big-endian)
//   Bytes 5+     : Payload    
//
// This package only ever touches bytes 0–4.

const (
	// HeaderSize is the number of bytes we need to read the service ID
	// and the request ID from the front of every incoming packet
	HeaderSize = 5 // 1 (ServiceID) + 4 (RequestID)
)

var (
	// ErrPacketTooShort means the incoming datagram didnt even have
	// enough bytes for us to read the header
	ErrPacketTooShort = errors.New("packet too short to contain a valid header")
)

// RequestHeader holds the two fields we extract from every packet
// The rest of the raw bytes pass will through untouched
type RequestHeader struct {
	ServiceID uint8
	RequestID uint32
}

// ParseHeader reads the service ID and request ID off the front of a raw
// UDP datagram
func ParseHeader(data []byte) (RequestHeader, error) {
	if len(data) < HeaderSize {
		return RequestHeader{}, ErrPacketTooShort
	}

	return RequestHeader{
		ServiceID: data[0],
		RequestID: binary.BigEndian.Uint32(data[1:5]),
	}, nil
}