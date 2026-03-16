package monitor

import (
	"net"
	"testing"
	"time"

	"bank-server/pkg/models"
)

// mockMarshal is a trivial marshaler that just returns a fixed byte slice.
// We don't care about the wire format in these unit tests — that's the
// handler layer's responsibility.
func mockMarshal(update models.AccountUpdate) ([]byte, error) {
	return []byte("mock"), nil
}

// newTestManager spins up a manager backed by a real loopback UDP socket.
// The sweeper interval is set deliberately high so it doesn't interfere
// with test timing unless we explicitly test sweeper behaviour.
func newTestManager(t *testing.T) (*Manager, *net.UDPConn) {
	t.Helper()

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0") // OS picks a free port
	if err != nil {
		t.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to listen on UDP: %v", err)
	}

	// Use a very long sweep interval so it doesn't fire during normal tests.
	mgr := NewManager(conn, mockMarshal, 1*time.Hour)

	t.Cleanup(func() {
		mgr.Stop()
		conn.Close()
	})

	return mgr, conn
}

func fakeClientAddr(t *testing.T, port int) *net.UDPAddr {
	t.Helper()
	return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
}

func TestManager_RegisterAndCount(t *testing.T) {
	mgr, _ := newTestManager(t)

	mgr.Register(fakeClientAddr(t, 9001), 30*time.Second)
	mgr.Register(fakeClientAddr(t, 9002), 60*time.Second)

	if count := mgr.ActiveCount(); count != 2 {
		t.Errorf("Expected 2 subscribers, got %d", count)
	}
}

func TestManager_RegisterOverwritesSameClient(t *testing.T) {
	mgr, _ := newTestManager(t)

	// Same client registers twice — the second call should overwrite, not duplicate.
	client := fakeClientAddr(t, 9001)
	mgr.Register(client, 10*time.Second)
	mgr.Register(client, 60*time.Second)

	if count := mgr.ActiveCount(); count != 1 {
		t.Errorf("Expected 1 subscriber after re-registration, got %d", count)
	}
}

func TestManager_LazyCleanupOnNotify(t *testing.T) {
	mgr, _ := newTestManager(t)

	// Register a subscriber that expires almost immediately
	mgr.Register(fakeClientAddr(t, 9001), 1*time.Millisecond)

	// Give it a moment to expire
	time.Sleep(10 * time.Millisecond)

	// NotifyAll should evict the expired subscriber during iteration
	update := models.AccountUpdate{
		ServiceID:     1,
		AccountNumber: 10000,
		HolderName:    "Test",
		CurrencyType:  models.SGD,
		NewBalance:    100.0,
	}
	mgr.NotifyAll(update)

	if count := mgr.ActiveCount(); count != 0 {
		t.Errorf("Expected 0 subscribers after lazy cleanup, got %d", count)
	}
}

func TestManager_PeriodicSweep(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Failed to listen on UDP: %v", err)
	}

	// Sweep every 50ms so we can observe it firing within the test window
	mgr := NewManager(conn, mockMarshal, 50*time.Millisecond)
	t.Cleanup(func() {
		mgr.Stop()
		conn.Close()
	})

	// Register a subscriber that expires in 10ms
	mgr.Register(fakeClientAddr(t, 9001), 10*time.Millisecond)

	// Wait enough time for the subscriber to expire and the sweeper to fire
	time.Sleep(150 * time.Millisecond)

	if count := mgr.ActiveCount(); count != 0 {
		t.Errorf("Expected sweeper to have cleaned up expired subscriber, got %d active", count)
	}
}

func TestManager_NotifySkipsExpiredKeepsActive(t *testing.T) {
	mgr, _ := newTestManager(t)

	// One subscriber that expires immediately, one that sticks around
	mgr.Register(fakeClientAddr(t, 9001), 1*time.Millisecond)
	mgr.Register(fakeClientAddr(t, 9002), 10*time.Minute)

	time.Sleep(10 * time.Millisecond)

	update := models.AccountUpdate{
		ServiceID:     3,
		AccountNumber: 10001,
		HolderName:    "Alice",
		CurrencyType:  models.USD,
		NewBalance:    500.0,
	}
	mgr.NotifyAll(update)

	if count := mgr.ActiveCount(); count != 1 {
		t.Errorf("Expected 1 active subscriber (expired one removed), got %d", count)
	}
}