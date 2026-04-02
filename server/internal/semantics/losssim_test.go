package semantics

import (
	"math"
	"testing"
)

func TestLossSimulator_ZeroRate_NeverDrops(t *testing.T) {
	ls := NewLossSimulator(0.0, 42)

	for i := 0; i < 1000; i++ {
		if ls.ShouldDrop() {
			t.Fatal("Zero-rate simulator should never drop packets")
		}
	}
}

func TestLossSimulator_FullRate_AlwaysDrops(t *testing.T) {
	ls := NewLossSimulator(1.0, 42)

	for i := 0; i < 1000; i++ {
		if !ls.ShouldDrop() {
			t.Fatal("Full-rate simulator should always drop packets")
		}
	}
}

func TestLossSimulator_StatisticalDistribution(t *testing.T) {
	// With a 30% drop rate over 10,000 trials, the observed rate
	// should be within a reasonable margin. We use ±5% tolerance
	// which is generous enough to avoid flaky tests.
	rate := 0.3
	trials := 10000
	ls := NewLossSimulator(rate, 12345)

	drops := 0
	for i := 0; i < trials; i++ {
		if ls.ShouldDrop() {
			drops++
		}
	}

	observedRate := float64(drops) / float64(trials)
	tolerance := 0.05

	if math.Abs(observedRate-rate) > tolerance {
		t.Errorf("Expected drop rate ~%.2f, observed %.4f (over %d trials): outside %.0f%% tolerance",
			rate, observedRate, trials, tolerance*100)
	}
}

func TestLossSimulator_ClampNegativeRate(t *testing.T) {
	ls := NewLossSimulator(-0.5, 42)

	if ls.Rate() != 0.0 {
		t.Errorf("Negative rate should be clamped to 0.0, got %f", ls.Rate())
	}

	if ls.IsEnabled() {
		t.Error("Simulator with clamped 0.0 rate should report as disabled")
	}
}

func TestLossSimulator_ClampExcessiveRate(t *testing.T) {
	ls := NewLossSimulator(1.5, 42)

	if ls.Rate() != 1.0 {
		t.Errorf("Rate above 1.0 should be clamped to 1.0, got %f", ls.Rate())
	}
}

func TestLossSimulator_IsEnabled(t *testing.T) {
	disabled := NewLossSimulator(0.0, 42)
	if disabled.IsEnabled() {
		t.Error("0.0 rate should be disabled")
	}

	enabled := NewLossSimulator(0.1, 42)
	if !enabled.IsEnabled() {
		t.Error("0.1 rate should be enabled")
	}
}

func TestLossSimulator_DeterministicWithSameSeed(t *testing.T) {
	// Two simulators with the same seed should produce identical sequences.
	// This is important for reproducible experiments in the lab report.
	ls1 := NewLossSimulator(0.5, 999)
	ls2 := NewLossSimulator(0.5, 999)

	for i := 0; i < 100; i++ {
		if ls1.ShouldDrop() != ls2.ShouldDrop() {
			t.Fatalf("Trial %d: same seed produced different results: determinism broken", i)
		}
	}
}
