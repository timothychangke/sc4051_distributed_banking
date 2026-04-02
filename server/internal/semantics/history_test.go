package semantics

import (
	"fmt"
	"sync"
	"testing"
)

func TestReplyHistory_StoreAndLookup(t *testing.T) {
	h := NewReplyHistory()

	reply := []byte{0x01, 0x02, 0x03}
	h.Store("127.0.0.1:9001", 1, reply)

	got, found := h.Lookup("127.0.0.1:9001", 1)
	if !found {
		t.Fatal("Expected to find cached reply, got nothing")
	}

	if len(got) != len(reply) {
		t.Fatalf("Expected reply of length %d, got %d", len(reply), len(got))
	}

	for i := range reply {
		if got[i] != reply[i] {
			t.Errorf("Byte %d: expected 0x%02x, got 0x%02x", i, reply[i], got[i])
		}
	}
}

func TestReplyHistory_LookupMiss(t *testing.T) {
	h := NewReplyHistory()

	// Nothing stored yet: everything should miss
	_, found := h.Lookup("127.0.0.1:9001", 1)
	if found {
		t.Error("Expected cache miss on empty history")
	}

	// Store something for a different client
	h.Store("127.0.0.1:9001", 1, []byte{0xFF})

	// Same client, different request ID
	_, found = h.Lookup("127.0.0.1:9001", 2)
	if found {
		t.Error("Expected cache miss for unknown request ID")
	}

	// Different client, same request ID
	_, found = h.Lookup("127.0.0.1:9002", 1)
	if found {
		t.Error("Expected cache miss for unknown client")
	}
}

func TestReplyHistory_DefensiveCopy(t *testing.T) {
	h := NewReplyHistory()

	// Store a reply, then mutate the original slice.
	// The cached copy should be unaffected.
	original := []byte{0x0A, 0x0B, 0x0C}
	h.Store("127.0.0.1:9001", 1, original)

	// Sabotage the original
	original[0] = 0xFF

	cached, _ := h.Lookup("127.0.0.1:9001", 1)
	if cached[0] != 0x0A {
		t.Errorf("Defensive copy failed: cached byte was mutated (got 0x%02x, expected 0x0A)", cached[0])
	}
}

func TestReplyHistory_EvictClient(t *testing.T) {
	h := NewReplyHistory()

	h.Store("127.0.0.1:9001", 1, []byte{0x01})
	h.Store("127.0.0.1:9001", 2, []byte{0x02})
	h.Store("127.0.0.1:9002", 1, []byte{0x03})

	h.EvictClient("127.0.0.1:9001")

	// Client 9001's entries should be gone
	if _, found := h.Lookup("127.0.0.1:9001", 1); found {
		t.Error("Expected client 9001's entries to be evicted")
	}

	// Client 9002 should be untouched
	if _, found := h.Lookup("127.0.0.1:9002", 1); !found {
		t.Error("Client 9002's entry should still exist after evicting 9001")
	}
}

func TestReplyHistory_EvictBefore(t *testing.T) {
	h := NewReplyHistory()

	client := "127.0.0.1:9001"
	h.Store(client, 1, []byte{0x01})
	h.Store(client, 2, []byte{0x02})
	h.Store(client, 5, []byte{0x05})
	h.Store(client, 10, []byte{0x0A})

	// Evict everything with ID < 5
	h.EvictBefore(client, 5)

	// IDs 1 and 2 should be gone
	if _, found := h.Lookup(client, 1); found {
		t.Error("Request ID 1 should have been evicted (< 5)")
	}
	if _, found := h.Lookup(client, 2); found {
		t.Error("Request ID 2 should have been evicted (< 5)")
	}

	// IDs 5 and 10 should survive
	if _, found := h.Lookup(client, 5); !found {
		t.Error("Request ID 5 should still exist (not < 5)")
	}
	if _, found := h.Lookup(client, 10); !found {
		t.Error("Request ID 10 should still exist (not < 5)")
	}
}

func TestReplyHistory_Size(t *testing.T) {
	h := NewReplyHistory()

	if h.Size() != 0 {
		t.Errorf("Expected size 0 on empty history, got %d", h.Size())
	}

	h.Store("client-a", 1, []byte{0x01})
	h.Store("client-a", 2, []byte{0x02})
	h.Store("client-b", 1, []byte{0x03})

	if h.Size() != 3 {
		t.Errorf("Expected size 3, got %d", h.Size())
	}
}

func TestReplyHistory_ConcurrentAccess(t *testing.T) {
	h := NewReplyHistory()
	var wg sync.WaitGroup

	writers := 100
	readersPerWriter := 5

	// Spawn 100 writers, each storing a unique (client, requestID) pair.
	// Simultaneously, spawn readers hammering Lookup on the same keys.
	// If the RWMutex is broken, the race detector will catch it.
	for i := 0; i < writers; i++ {
		client := fmt.Sprintf("127.0.0.1:%d", 9000+i)
		reqID := uint32(i)

		// Writer
		wg.Add(1)
		go func(c string, id uint32) {
			defer wg.Done()
			h.Store(c, id, []byte{byte(id)})
		}(client, reqID)

		// Readers
		for j := 0; j < readersPerWriter; j++ {
			wg.Add(1)
			go func(c string, id uint32) {
				defer wg.Done()
				// We don't care about the result: we're testing for panics
				// and data races, not correctness of intermediate reads.
				_, _ = h.Lookup(c, id)
			}(client, reqID)
		}
	}

	wg.Wait()

	// After all goroutines finish, we should have exactly 100 entries
	if h.Size() != writers {
		t.Errorf("Expected %d entries after concurrent writes, got %d", writers, h.Size())
	}
}
