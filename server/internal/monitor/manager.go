package monitor

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"bank-server/pkg/models"
)

// The subscriber holds everything we need to push updates to a registered client.
// Clients address comes straight from the UDP packets source
type subscriber struct {
	addr      *net.UDPAddr
	expiresAt time.Time
}

// Function signature that the handler layer provides
// to convert an AccountUpdate into raw bytes
// (Need to implement)
type MarshalUpdateFunc func(update models.AccountUpdate) ([]byte, error)

// Manager tracks all clients that have registered for callback monitoring
// It is safe for concurrent use as both the main request and background sweeper use
// the subscriber map 
type Manager struct {
	mu          sync.Mutex
	subscribers map[string]*subscriber 

	conn      *net.UDPConn     // main UDP socket for sending callbacks
	marshaler MarshalUpdateFunc // injected by the handler layer at startup

	sweepInterval time.Duration // how often the background goroutine checks for expired subscribers
	stopSweep     chan struct{} // calls the sweeper goroutine to shut down
}

// NewManager creates a monitor manager and starts the periodic sweeper
func NewManager(conn *net.UDPConn, marshaler MarshalUpdateFunc, sweepInterval time.Duration) *Manager {
	m := &Manager{
		subscribers:   make(map[string]*subscriber),
		conn:          conn,
		marshaler:     marshaler,
		sweepInterval: sweepInterval,
		stopSweep:     make(chan struct{}),
	}

	go m.periodicSweep()

	return m
}

// Register adds a client to the subscriber list. If client is already in monitoring,
// it overwrites the previous registration and updates its interval
func (m *Manager) Register(clientAddr *net.UDPAddr, interval time.Duration) {
	key := clientAddr.String()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscribers[key] = &subscriber{
		addr:      clientAddr,
		expiresAt: time.Now().Add(interval),
	}

	log.Printf("[Monitor] Registered subscriber %s for %v (expires at %s)",
		key, interval, m.subscribers[key].expiresAt.Format(time.RFC3339))
}

// This function pushes the update to all subscribers
// For expired subscribers, they are lazily cleaned up during the iteratio
// There is also a periodic sweep of clients to check if they are expired
func (m *Manager) NotifyAll(update models.AccountUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.subscribers) == 0 {
		return
	}

	now := time.Now()

	// Marshal only once
	payload, err := m.marshaler(update)
	if err != nil {
		log.Printf("[Monitor] Failed to marshal update: %v", err)
		return
	}

	for key, sub := range m.subscribers {
		// If interval has expired, evict
		if now.After(sub.expiresAt) {
			log.Printf("[Monitor] Subscriber %s expired, removing", key)
			delete(m.subscribers, key)
			continue
		}

		if _, err := m.conn.WriteToUDP(payload, sub.addr); err != nil {
			log.Printf("[Monitor] Failed to push update to %s: %v", key, err)
		} else {
			log.Printf("[Monitor] Pushed update (service=%d, acc=%d) to %s",
				update.ServiceID, update.AccountNumber, key)
		}
	}
}

// Returns how many active subscribers there are
func (m *Manager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.subscribers)
}

// Stops background sweeper goroutine
func (m *Manager) Stop() {
	close(m.stopSweep)
}

// Background Goroutine that removes expired subscribers at an interval
// This catches subscribers that have expired but when there are no mutations happening
func (m *Manager) periodicSweep() {
	ticker := time.NewTicker(m.sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopSweep:
			log.Println("[Monitor] Sweeper shutting down")
			return
		case <-ticker.C:
			m.evictExpired()
		}
	}
}

// Walks the subscriber map and remove all clients that are past expiry
func (m *Manager) evictExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, sub := range m.subscribers {
		if now.After(sub.expiresAt) {
			fmt.Printf("[Monitor] Sweeper removing expired subscriber %s\n", key)
			delete(m.subscribers, key)
		}
	}
}