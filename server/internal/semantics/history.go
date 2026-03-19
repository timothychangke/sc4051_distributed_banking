package semantics

import (
	"log"
	"sync"
)

// This is a server side cache of replies weve already sent.
//
// In at-most-once semantics, if we get a duplicate request (same client,
// same request ID), we don't re-execute the handler — we just re-send
// the cached reply. This prevents non-idempotent operations like Transfer
// from executing twice when a reply gets lost and the client retransmits.
//
// The outer map is keyed by the client's ip:port string, the inner map
// is keyed by the request ID
type ReplyHistory struct {
	mu      sync.RWMutex
	entries map[string]map[uint32][]byte
}

// Creates an empty  history cache
func NewReplyHistory() *ReplyHistory {
	return &ReplyHistory{
		entries: make(map[string]map[uint32][]byte),
	}
}

// Checks whether we have already processed and replied to this exact
// request. Returns the cached reply bytes and true if found, or nil and
// false if this is a fresh request we havent seen before
func (h *ReplyHistory) Lookup(clientAddr string, requestID uint32) ([]byte, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientEntries, exists := h.entries[clientAddr]
	if !exists {
		return nil, false
	}

	reply, found := clientEntries[requestID]
	return reply, found
}

// Store saves the raw reply bytes so we can re-send them if the client
// retransmits the same request. This should be called right after we
// get a reply from the handler and before we send it over the wire.
func (h *ReplyHistory) Store(clientAddr string, requestID uint32, reply []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.entries[clientAddr]; !exists {
		h.entries[clientAddr] = make(map[uint32][]byte)
	}

	// Make a defensive copy so the caller cant mutate our cached data.
	stored := make([]byte, len(reply))
	copy(stored, reply)
	h.entries[clientAddr][requestID] = stored
}

// EvictClient removes all cached replies for a specific client for clean up after client disconnects.
func (h *ReplyHistory) EvictClient(clientAddr string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.entries, clientAddr)
	log.Printf("[ReplyHistory] Evicted all entries for client %s", clientAddr)
}

// EvictBefore removes all cached entries for a client whose request ID
// is strictly less than the given threshold. Once the client sends request N,
// we know that it will never retransmit anything older than N minus some window.
func (h *ReplyHistory) EvictBefore(clientAddr string, threshold uint32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clientEntries, exists := h.entries[clientAddr]
	if !exists {
		return
	}

	evicted := 0
	for id := range clientEntries {
		if id < threshold {
			delete(clientEntries, id)
			evicted++
		}
	}

	if evicted > 0 {
		log.Printf("[ReplyHistory] Evicted %d stale entries for client %s (threshold: %d)",
			evicted, clientAddr, threshold)
	}
}

// Returns the total number of cached replies across all clients.
func (h *ReplyHistory) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clientEntries := range h.entries {
		total += len(clientEntries)
	}
	return total
}