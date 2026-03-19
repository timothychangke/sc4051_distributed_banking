package semantics

import (
	"encoding/binary"
	"net"
	"sync/atomic"
	"testing"
)

// buildTestPacket creates a minimal valid request packet with the given
// service ID and request ID. The payload is empty because the dispatcher
// never looks past byte 4 — that's the marshalling layer's job.
func buildTestPacket(serviceID uint8, requestID uint32) []byte {
	data := make([]byte, HeaderSize)
	data[0] = serviceID
	binary.BigEndian.PutUint32(data[1:5], requestID)
	return data
}

func fakeAddr(port int) *net.UDPAddr {
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
}

// TestDispatcher_AtLeastOnce_AlwaysExecutes proves that at-least-once
// mode calls the handler every single time, even for duplicate requests.
// This is the behavior that causes double-deposits in lossy networks.
func TestDispatcher_AtLeastOnce_AlwaysExecutes(t *testing.T) {
	var callCount atomic.Int32

	handler := func(data []byte, addr *net.UDPAddr) []byte {
		callCount.Add(1)
		return []byte{0x01} // dummy reply
	}

	d := NewDispatcher(AtLeastOnce, handler)
	addr := fakeAddr(9001)
	packet := buildTestPacket(3, 1) // service=Deposit, requestID=1

	// Send the same request 5 times (simulating 4 retransmissions)
	for i := 0; i < 5; i++ {
		reply, err := d.Dispatch(packet, addr)
		if err != nil {
			t.Fatalf("Dispatch %d: unexpected error: %v", i, err)
		}
		if reply == nil {
			t.Fatalf("Dispatch %d: expected non-nil reply", i)
		}
	}

	// The handler should have been invoked all 5 times.
	// This is the dangerous part — if this were a $100 deposit,
	// the account just got $500 credited.
	if callCount.Load() != 5 {
		t.Errorf("Expected handler to be called 5 times (at-least-once), got %d", callCount.Load())
	}
}

// TestDispatcher_AtMostOnce_DeduplicatesDuplicates proves that at-most-once
// mode only executes the handler once for a given (client, requestID) pair.
// Subsequent retransmissions get the cached reply without touching the handler.
func TestDispatcher_AtMostOnce_DeduplicatesDuplicates(t *testing.T) {
	var callCount atomic.Int32

	expectedReply := []byte{0xAA, 0xBB}
	handler := func(data []byte, addr *net.UDPAddr) []byte {
		callCount.Add(1)
		return expectedReply
	}

	d := NewDispatcher(AtMostOnce, handler)
	addr := fakeAddr(9001)
	packet := buildTestPacket(7, 1) // service=Transfer, requestID=1

	// First dispatch — should execute the handler
	reply1, err := d.Dispatch(packet, addr)
	if err != nil {
		t.Fatalf("First dispatch: unexpected error: %v", err)
	}

	// Send the same request 4 more times (retransmissions)
	for i := 0; i < 4; i++ {
		reply, err := d.Dispatch(packet, addr)
		if err != nil {
			t.Fatalf("Dispatch %d: unexpected error: %v", i+1, err)
		}

		// The cached reply should be byte-identical to the first one
		if len(reply) != len(reply1) {
			t.Errorf("Dispatch %d: reply length mismatch", i+1)
		}
		for j := range reply {
			if reply[j] != reply1[j] {
				t.Errorf("Dispatch %d: reply byte %d differs from original", i+1, j)
			}
		}
	}

	// The handler should have been called exactly once.
	// The Transfer only executes once — that's the whole point.
	if callCount.Load() != 1 {
		t.Errorf("Expected handler to be called exactly 1 time (at-most-once), got %d", callCount.Load())
	}
}

// TestDispatcher_AtMostOnce_DifferentRequestIDs proves that different request
// IDs from the same client are treated as distinct requests (not duplicates).
func TestDispatcher_AtMostOnce_DifferentRequestIDs(t *testing.T) {
	var callCount atomic.Int32

	handler := func(data []byte, addr *net.UDPAddr) []byte {
		callCount.Add(1)
		return []byte{0x01}
	}

	d := NewDispatcher(AtMostOnce, handler)
	addr := fakeAddr(9001)

	// 3 different request IDs from the same client
	for reqID := uint32(1); reqID <= 3; reqID++ {
		packet := buildTestPacket(6, reqID)
		_, err := d.Dispatch(packet, addr)
		if err != nil {
			t.Fatalf("Request %d: unexpected error: %v", reqID, err)
		}
	}

	if callCount.Load() != 3 {
		t.Errorf("Expected 3 handler calls for 3 unique request IDs, got %d", callCount.Load())
	}
}

// TestDispatcher_AtMostOnce_DifferentClients proves that the same request ID
// from different clients are independent (client A's request 1 ≠ client B's request 1).
func TestDispatcher_AtMostOnce_DifferentClients(t *testing.T) {
	var callCount atomic.Int32

	handler := func(data []byte, addr *net.UDPAddr) []byte {
		callCount.Add(1)
		return []byte{0x01}
	}

	d := NewDispatcher(AtMostOnce, handler)
	packet := buildTestPacket(3, 1) // same service, same request ID

	// Two different clients sending the same request ID
	_, _ = d.Dispatch(packet, fakeAddr(9001))
	_, _ = d.Dispatch(packet, fakeAddr(9002))

	if callCount.Load() != 2 {
		t.Errorf("Expected 2 handler calls for 2 different clients, got %d", callCount.Load())
	}
}

// TestDispatcher_MalformedPacket verifies that a packet too short to
// contain a header returns an error without panicking or calling the handler.
func TestDispatcher_MalformedPacket(t *testing.T) {
	handler := func(data []byte, addr *net.UDPAddr) []byte {
		t.Fatal("Handler should never be called for a malformed packet")
		return nil
	}

	d := NewDispatcher(AtMostOnce, handler)

	_, err := d.Dispatch([]byte{0x01}, fakeAddr(9001))
	if err == nil {
		t.Error("Expected error for malformed packet, got nil")
	}
}

// TestDispatcher_AtMostOnce_HistoryAccessible verifies the History()
// accessor returns the underlying cache for diagnostics.
func TestDispatcher_AtMostOnce_HistoryAccessible(t *testing.T) {
	d := NewDispatcher(AtMostOnce, func(data []byte, addr *net.UDPAddr) []byte {
		return []byte{0x01}
	})

	if d.History() == nil {
		t.Error("Expected non-nil history in at-most-once mode")
	}

	// Process one request so the cache has an entry
	packet := buildTestPacket(1, 1)
	_, _ = d.Dispatch(packet, fakeAddr(9001))

	if d.History().Size() != 1 {
		t.Errorf("Expected 1 cached entry, got %d", d.History().Size())
	}
}

// TestDispatcher_AtLeastOnce_NoHistory verifies that at-least-once mode
// doesn't waste memory on a reply cache it'll never use.
func TestDispatcher_AtLeastOnce_NoHistory(t *testing.T) {
	d := NewDispatcher(AtLeastOnce, func(data []byte, addr *net.UDPAddr) []byte {
		return []byte{0x01}
	})

	if d.History() != nil {
		t.Error("Expected nil history in at-least-once mode")
	}
}