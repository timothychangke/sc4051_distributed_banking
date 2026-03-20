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
	// HeaderSize is the number of bytes we need to read the full C++ client
	// message header from every incoming packet.
	HeaderSize = 18
)

var (
	// ErrPacketTooShort means the incoming datagram didnt even have
	// enough bytes for us to read the header
	ErrPacketTooShort = errors.New("packet too short to contain a valid header")
)

// MessageHeader holds the fields we extract from the 18-byte client header.
type MessageHeader struct {
	Type        uint8
	Flag        uint8
	RequestID   uint32
	IPv4        uint32
	Port        uint16
	StatusCode  uint16
	ContentLen  uint32
}

// ParseHeader reads the fixed-size message header off the front of a raw
// UDP datagram. This matches the C++ Protocol::MessageSerializer::serialize()
// format exactly.
func ParseHeader(data []byte) (MessageHeader, error) {
	if len(data) < HeaderSize {
		return MessageHeader{}, ErrPacketTooShort
	}

	return MessageHeader{
		Type:       data[0],
		Flag:       data[1],
		RequestID:  binary.BigEndian.Uint32(data[2:6]),
		IPv4:       binary.BigEndian.Uint32(data[6:10]),
		Port:       binary.BigEndian.Uint16(data[10:12]),
		StatusCode: binary.BigEndian.Uint16(data[12:14]),
		ContentLen: binary.BigEndian.Uint32(data[14:18]),
	}, nil
}