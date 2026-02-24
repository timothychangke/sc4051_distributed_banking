package marshal

// Encoder writes into a byte slice
type Encoder struct {
	buf []byte
}

func NewEncoder() *Encoder {
	// Initialize the buffer as an empty byte slice
	return &Encoder{
		buf: make([]byte, 0),
	}
}

// Decoder reads from a byte slice
type Decoder struct {
	buf    []byte
	offset int
}

func NewDecoder(data []byte) *Decoder {
	return &Decoder{
		buf:    data,
		offset: 0,
	}
}
