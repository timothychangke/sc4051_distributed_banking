package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"bank-server/internal/banking"
	"bank-server/internal/handler"
	marshal "bank-server/internal/marshalling"
	"bank-server/internal/monitor"
	"bank-server/internal/semantics"
	"bank-server/internal/store"
)

// ─────────────────────────────────────────────────────────────────────
// Distributed Banking Server
//
// Usage:
//   ./server <at-least-once|at-most-once> [loss-rate]
//
// Examples:
//   ./server at-most-once           # no simulated loss
//   ./server at-least-once 0.3      # 30% reply loss
//   ./server at-most-once 0.5       # 50% reply loss
//
// The loss rate is a float between 0.0 and 1.0 that controls how often
// replies are silently dropped. This is how we provoke retransmissions
// for the fault tolerance experiment.
// ─────────────────────────────────────────────────────────────────────

const (
	defaultPort         = 2222
	defaultSweepSeconds = 5
	maxPacketSize       = 4096
	lossSeed            = 42 // fixed seed so experiments are reproducible
)

func main() {
	// ──────────────────────────────────────────────────────
	// 0. PARSE CLI ARGUMENTS
	// ──────────────────────────────────────────────────────
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <at-least-once|at-most-once> [loss-rate]\n", os.Args[0])
		os.Exit(1)
	}

	mode, err := semantics.ParseMode(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	lossRate := parseLossRate(os.Args)

	log.Printf("[Server] Invocation semantics: %s", mode)
	log.Printf("[Server] Simulated reply loss rate: %.0f%%", lossRate*100)

	// ──────────────────────────────────────────────────────
	// 1. UDP SOCKET SETUP
	// ──────────────────────────────────────────────────────
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: defaultPort}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("[Server] Failed to bind UDP on port %d: %v", defaultPort, err)
	}
	defer conn.Close()

	log.Printf("[Server] Listening on %s", conn.LocalAddr())

	// ──────────────────────────────────────────────────────
	// 2. DEPENDENCY WIRING
	// ──────────────────────────────────────────────────────
	memStore := store.NewMemoryStore()
	bankingSvc := banking.NewService(memStore)

	// ──────────────────────────────────────────────────────
	// 3. CALLBACK MONITOR SETUP
	//
	// The monitor manager reuses the server's UDP socket to push
	// callback packets to subscribed clients. MarshalCallbackUpdate
	// is injected so the monitor layer stays decoupled from wire format.
	// ──────────────────────────────────────────────────────
	sweepInterval := defaultSweepSeconds * time.Second
	monitorMgr := monitor.NewManager(conn, marshal.MarshalCallbackUpdate, sweepInterval)
	defer monitorMgr.Stop()

	// ──────────────────────────────────────────────────────
	// 4. HANDLER + INVOCATION SEMANTICS
	// ──────────────────────────────────────────────────────
	requestHandler := handler.BuildHandler(bankingSvc, monitorMgr)
	dispatcher := semantics.NewDispatcher(mode, requestHandler)
	replyLoss := semantics.NewLossSimulator(lossRate, lossSeed)

	// ──────────────────────────────────────────────────────
	// 5. MAIN REQUEST LOOP
	//
	// Reads datagrams, runs them through the dispatcher (which
	// handles duplicate filtering in at-most-once mode), then
	// sends the reply — unless the loss simulator says to drop it.
	// ──────────────────────────────────────────────────────
	log.Println("[Server] Ready to accept requests")

	buf := make([]byte, maxPacketSize)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("[Server] ReadFromUDP error: %v", err)
			continue
		}
		log.Printf("[Server] Received %d bytes from %s", n, clientAddr)

		reply, err := dispatcher.Dispatch(buf[:n], clientAddr)
		if err != nil {
			log.Printf("[Server] Dispatch error: %v", err)
			continue
		}

		// Simulate reply loss for fault tolerance experiments.
		// When the reply is dropped, the client times out and retransmits.
		// Under at-most-once, the dispatcher returns the cached reply.
		// Under at-least-once, the handler re-executes (which is the bug
		// we want to demonstrate for non-idempotent ops).
		if replyLoss.ShouldDrop() {
			log.Printf("[Server] Simulated reply loss for request from %s", clientAddr)
			continue
		}

		if _, err := conn.WriteToUDP(reply, clientAddr); err != nil {
			log.Printf("[Server] WriteToUDP error: %v", err)
		} else {
			log.Printf("[Server] Sent %d byte reply to %s", len(reply), clientAddr)
		}
	}
}

// parseLossRate extracts the optional loss rate from CLI args.
// Returns 0.0 if not provided or invalid.
func parseLossRate(args []string) float64 {
	if len(args) < 3 {
		return 0.0
	}

	rate, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		log.Printf("[Server] Warning: invalid loss rate %q, defaulting to 0.0", args[2])
		return 0.0
	}

	if rate < 0.0 || rate > 1.0 {
		log.Printf("[Server] Warning: loss rate %.2f out of range [0.0, 1.0], clamping", rate)
		if rate < 0.0 {
			rate = 0.0
		}
		if rate > 1.0 {
			rate = 1.0
		}
	}

	return rate
}