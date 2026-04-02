package semantics

import (
	"fmt"
	"log"
	"net"
)

// Mode represents which invocation semantic the server is running under
// This gets set once at startup from a CLI flag and never changes
type Mode uint8

const (
	None Mode = iota // Matches client default of 0
	AtLeastOnce
	AtMostOnce
)

// String makes Mode print nicely in logs instead of showing "0" or "1"
func (m Mode) String() string {
	switch m {
	case AtLeastOnce:
		return "at-least-once"
	case AtMostOnce:
		return "at-most-once"
	default:
		return "none"
	}
}

// ParseMode converts a CLI string into a Mode value
// Accepts "at-least-once" and "at-most-once":
func ParseMode(s string) (Mode, error) {
	switch s {
	case "at-least-once":
		return AtLeastOnce, nil
	case "at-most-once":
		return AtMostOnce, nil
	default:
		return None, fmt.Errorf("unknown invocation semantics mode: %q (expected \"at-least-once\" or \"at-most-once\")", s)
	}
}

// Function signature to NOTE FOR ZHIXUAN It takes raw request bytes and the client's address, does
// the banking logic, and returns raw reply bytes.
//
// The dispatcher doesnt care whats inside these bytes beyond the header.
// This is the boundary between your work and the marshalling layer.
type RequestHandler func(data []byte, addr *net.UDPAddr) []byte

// Dispatcher sits between the UDP read loop and the actual request handler.
// Its the only component that changes behavior based on invocation semantics.
//
// In at-least-once mode, every request hits the handler. Simple, but dangerous
// for non-idempotent ops if the client retransmits.
//
// In at-most-once mode, duplicate requests get the cached reply instead of
// re-executing. This is what makes Transfer safe under message loss.
type Dispatcher struct {
	mode    Mode
	handler RequestHandler
	history *ReplyHistory
}

// NewDispatcher creates a dispatcher wired to the given handler.
// In at-least-once mode, the history is unused (nil internally).
// In at-most-once mode, a fresh ReplyHistory is allocated automatically.
func NewDispatcher(mode Mode, handler RequestHandler) *Dispatcher {
	d := &Dispatcher{
		mode:    mode,
		handler: handler,
	}

	if mode == AtMostOnce {
		d.history = NewReplyHistory()
	}

	log.Printf("[Dispatcher] Initialized with %s semantics", mode)
	return d
}

// Dispatch processes an incoming request packet. This is what the server
// loop calls for every datagram it reads off the wire
//
// Returns the reply bytes to send back to the client
// Returns an error only if the packet header itself is wrong
func (d *Dispatcher) Dispatch(data []byte, addr *net.UDPAddr) ([]byte, error) {
	header, err := ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request header: %w", err)
	}

	if Mode(header.Flag) != d.mode {
		return nil, fmt.Errorf("request flag %d does not match dispatcher mode %d", header.Flag, d.mode)
	}

	clientKey := addr.String()

	log.Printf("[Dispatcher] Received request: requestID=%d, client=%s, mode=%s",
		header.RequestID, clientKey, d.mode)

	// At-least-once: always execute. It does no filtering and no caching.
	// If the client retransmits, we just run the operation again.
	// This is fine for idempotent ops but will cause double-deposits
	// and double-transfers if a reply gets lost
	if d.mode == AtLeastOnce {
		reply := d.handler(data, addr)
		log.Printf("[Dispatcher] At-least-once: executed request %d from %s",
			header.RequestID, clientKey)
		return reply, nil
	}

	// At-most-once: check the cache before doing anything.
	// If weve seen this exact (client, requestID) before, the client
	// is retransmitting because our previous reply got lost. Just
	// re-send the cached reply: do NOT re-execute the handler.
	if cachedReply, found := d.history.Lookup(clientKey, header.RequestID); found {
		log.Printf("[Dispatcher] At-most-once: duplicate detected for request %d from %s, returning cached reply",
			header.RequestID, clientKey)
		return cachedReply, nil
	}

	// First time seeing this request, execute normally.
	reply := d.handler(data, addr)

	// Cache the reply before sending so we can re-serve it on retransmits
	d.history.Store(clientKey, header.RequestID, reply)

	log.Printf("[Dispatcher] At-most-once: executed and cached request %d from %s",
		header.RequestID, clientKey)
	return reply, nil
}

// History exposes the underlying ReplyHistory for testing and diagnostics
// Returns nil if the dispatcher is running in at-least-once mode
func (d *Dispatcher) History() *ReplyHistory {
	return d.history
}

// Mode returns the invocation semantics this dispatcher is operating under
func (d *Dispatcher) Mode() Mode {
	return d.mode
}
