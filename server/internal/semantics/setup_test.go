package semantics

import (
	"io"
	"log"
	"os"
	"testing"
)

// TestMain silences the standard logger during test runs so the output
// isn't drowned in [Dispatcher], [LossSimulator], and [ReplyHistory] noise.
// The tests validate behavior through return values and assertions, not logs.
//
// If you ever need to debug a failing test, just comment out the SetOutput
// line and the full log stream comes back.
func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}