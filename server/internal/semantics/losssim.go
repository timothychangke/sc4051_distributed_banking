package semantics

import (
	"log"
	"math/rand"
)

// This simulator willl randomly drops packets to simulate an unreliable network.
// This is how we test that at-most-once actually prevents duplicate execution
// and that at-least-once causes the problems the lab report needs to discuss.
//
// is called before sending a reply ShouldDrop(). If it returns
// true, skip the WriteToUDP — the client will time out and retransmit,
// which is exactly the scenario we want to provoke.
type LossSimulator struct {
	rate float64
	rng  *rand.Rand
}

// Creates a simulator with the given drop probability.
// rate should be between 0.0 (never drop) and 1.0 (always drop).
//
// We use a dedicated rand.Rand source instead of the global rand so that
// concurrent callers dont contend on the global lock. 
func NewLossSimulator(rate float64, seed int64) *LossSimulator {
	// Clamp the rate to valid bounds so a typo doesnt break things
	if rate < 0.0 {
		rate = 0.0
	}
	if rate > 1.0 {
		rate = 1.0
	}

	return &LossSimulator{
		rate: rate,
		rng:  rand.New(rand.NewSource(seed)),
	}
}

// ShouldDrop decides whether this packet should be
// lost in transit. The caller is responsible for actually skipping the
// send — we just make the decision.
func (ls *LossSimulator) ShouldDrop() bool {
	if ls.rate == 0.0 {
		return false
	}

	drop := ls.rng.Float64() < ls.rate
	if drop {
		log.Println("[LossSimulator] Packet dropped (simulated loss)")
	}
	return drop
}

// Rate returns the configured drop probability. 
func (ls *LossSimulator) Rate() float64 {
	return ls.rate
}

// IsEnabled returns true if the simulator will actually drop packets.
// A rate of 0.0 means it's effectively a no-op pass-through.
func (ls *LossSimulator) IsEnabled() bool {
	return ls.rate > 0.0
}