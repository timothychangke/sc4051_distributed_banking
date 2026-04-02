// integration_test.go: Full integration test suite for the Distributed Banking System.
//
// This file drives the real C++ client binary against the real Go server binary,
// piping commands through stdin and asserting on the stdout output.
//
// HOW IT WORKS:
//   1. Each test spins up a fresh Go server on port 2222 (the hardcoded default).
//   2. It launches the C++ client via a PTY wrapper (script/stdbuf) so stdout isn't
//      block-buffered: this is critical because the C++ client uses std::cout without flush.
//   3. Commands are fed through stdin; responses are read from stdout.
//   4. Assertions check that the client printed the expected results.
//
// IMPORTANT: Because the server hardcodes port 2222, tests run SEQUENTIALLY.
// A global mutex prevents overlapping server instances.
//
// PREREQUISITES:
//   - Go server binary:   go build -o ./bin/server ./cmd/main.go
//   - C++ client binary:  (your makefile) -> ./bin/client
//
// RUN:
//   go test -v -timeout 600s -count=1 ..
//
// OVERRIDE PATHS:
//   SERVER_BIN=./bin/server CLIENT_BIN=./bin/client go test -v .

package integration_test

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Configuration: update these paths to match your build output locations.
// ---------------------------------------------------------------------------

func serverBin() string {
	if v := os.Getenv("SERVER_BIN"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		return "../server/bin/server.exe"
	}
	return "../server/bin/server"
}

func clientBin() string {
	if v := os.Getenv("CLIENT_BIN"); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		return "../client/build/Debug/client.exe"
	}
	return "../client/build/client"
}

// The server hardcodes port 2222, so every test must use this port.
// A global mutex ensures only one server runs at a time.
const serverPort = 2222

var serverMu sync.Mutex

func TestMain(m *testing.M) {
	// Best-effort cleanup: if nothing is on the port, the command silently fails.
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		kill := exec.Command("sh", "-c", "kill -9 $(lsof -ti :2222) 2>/dev/null")
		_ = kill.Run()
		// Give the OS a moment to release the socket.
		time.Sleep(500 * time.Millisecond)
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Menu option constants: these MUST match your C++ client's menu.
//
// Your actual client menu (from bankIO.cpp):
//   1 = OPEN       3 = DEPOSIT     7 = TRANSFER
//   2 = CLOSE      4 = WITHDRAW    0 = EXIT
//   5 = MONITOR    6 = BALANCE
// ---------------------------------------------------------------------------

const (
	MenuOpenAccount  = "1"
	MenuCloseAccount = "2"
	MenuDeposit      = "3"
	MenuWithdraw     = "4"
	MenuMonitor      = "5"
	MenuGetBalance   = "6"
	MenuTransfer     = "7"
	MenuExit         = "0"
)

// Your C++ client expects currency as uppercase strings, not numbers.
const (
	CurrSGD = "SGD"
	CurrUSD = "USD"
	CurrEUR = "EUR"
)

const (
	SemanticsAtLeastOnce = "-l"
	SemanticsAtMostOnce  = "-m"
)

// How long we'll wait for a response before declaring timeout.
const defaultTimeout = 8 * time.Second

// ANSI escape code stripper: the client uses color codes and clear_ui() escape sequences.
// We need to strip these so our string matching works on clean text.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\[\?[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// ---------------------------------------------------------------------------
// ServerProcess manages the lifecycle of a Go server subprocess.
// ---------------------------------------------------------------------------

type ServerProcess struct {
	cmd    *exec.Cmd
	mode   string
	stdout io.ReadCloser
	stderr io.ReadCloser
	outBuf *lineBuffer
	errBuf *lineBuffer
}

// startServer launches the Go server and waits for it to bind.
// Caller MUST hold serverMu or call this from within a test that serializes.
func startServer(t *testing.T, mode string, extraArgs ...string) *ServerProcess {
	t.Helper()

	args := []string{mode}
	args = append(args, extraArgs...)

	cmd := exec.Command(serverBin(), args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to pipe server stdout: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to pipe server stderr: %v", err)
	}

	sp := &ServerProcess{
		cmd:    cmd,
		mode:   mode,
		stdout: stdout,
		stderr: stderr,
		outBuf: newLineBuffer(),
		errBuf: newLineBuffer(),
	}

	go sp.outBuf.drain(stdout)
	go sp.errBuf.drain(stderr)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Wait until the server has bound the UDP socket and printed its startup line.
	// We look for "Listening" in the log output as the ready signal.
	ready := sp.waitForOutput("Listening", 8*time.Second)
	if !ready {
		time.Sleep(2 * time.Second)
	}

	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)

		// Give it a moment to shut down gracefully.
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()

		select {
		case <-done:
			// Clean exit.
		case <-time.After(2 * time.Second):
			// Force kill if it didn't respond to interrupt.
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}

		// Wait for the OS to release the UDP port.
		time.Sleep(500 * time.Millisecond)
	})

	return sp
}

func (sp *ServerProcess) outputContains(substr string) bool {
	return sp.outBuf.contains(substr) || sp.errBuf.contains(substr)
}

func (sp *ServerProcess) waitForOutput(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if sp.outputContains(substr) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func (sp *ServerProcess) allOutput() string {
	return sp.outBuf.snapshot() + "\n" + sp.errBuf.snapshot()
}

// ---------------------------------------------------------------------------
// ClientProcess manages the C++ client subprocess.
//
// THE KEY INSIGHT: When stdout is piped (not a TTY), the C++ standard library
// uses full buffering on std::cout. Your client never calls std::flush, so
// the test sees ZERO output until the process exits or the 4KB buffer fills.
//
// We fix this by launching the client through a PTY wrapper:
//   macOS:  script -q /dev/null ./client <args>
//   Linux:  stdbuf -oL ./client <args>
//
// This forces line-buffered or unbuffered stdout so we can read responses
// in real time as the client prints them.
// ---------------------------------------------------------------------------

type ClientProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	outBuf *lineBuffer
}

func startClient(t *testing.T, extraArgs ...string) *ClientProcess {
	t.Helper()

	clientArgs := []string{"127.0.0.1", fmt.Sprintf("%d", serverPort)}
	clientArgs = append(clientArgs, extraArgs...)

	// Build the actual command with a PTY/unbuffer wrapper.
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command(clientBin(), clientArgs...)
	case "linux":
		// Linux: `stdbuf -oL` forces line-buffered stdout.
		stdbufArgs := []string{"-oL", clientBin()}
		stdbufArgs = append(stdbufArgs, clientArgs...)
		cmd = exec.Command("stdbuf", stdbufArgs...)
	default:
		// Fallback: run directly and hope for the best.
		cmd = exec.Command(clientBin(), clientArgs...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to pipe client stdin: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to pipe client stdout: %v", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into the same stream

	cp := &ClientProcess{
		cmd:    cmd,
		stdin:  stdin,
		outBuf: newLineBuffer(),
	}

	go cp.outBuf.drain(stdout)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start client: %v", err)
	}

	// Wait for the menu to appear: this confirms the client started and stdout is flowing.
	if !cp.waitForOutput("SYSTEM", 5*time.Second) {
		t.Fatalf("client did not print menu within 5s.\nCaptured output:\n%s", cp.outputSnapshot())
	}

	t.Cleanup(func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	return cp
}

func (cp *ClientProcess) send(line string) {
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	_, _ = io.WriteString(cp.stdin, line)
}

// sendLines writes multiple lines with a small delay between each.
// The C++ client reads some fields with `read_int()` (cin >>) and others with
// `read_line()` (getline). The delay ensures each value is consumed before the next.
func (cp *ClientProcess) sendLines(lines ...string) {
	for _, l := range lines {
		cp.send(l)
		time.Sleep(150 * time.Millisecond)
	}
}

// pressEnter sends a bare newline to satisfy `wait_for_enter()` calls.
// The C++ client calls this after every operation: cin.ignore() + cin.get().
func (cp *ClientProcess) pressEnter() {
	cp.send("") // consumed by cin.ignore()
	cp.send("") // consumed by cin.get()
	time.Sleep(300 * time.Millisecond)
}

func (cp *ClientProcess) waitForOutput(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cp.outBuf.containsClean(substr) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func (cp *ClientProcess) outputContains(substr string) bool {
	return cp.outBuf.containsClean(substr)
}

// outputSnapshot returns the ANSI-stripped, cleaned output for assertions and debugging.
func (cp *ClientProcess) outputSnapshot() string {
	return cp.outBuf.cleanSnapshot()
}

func (cp *ClientProcess) clearOutput() {
	cp.outBuf.clear()
}

// ---------------------------------------------------------------------------
// lineBuffer: thread-safe text accumulator that also strips ANSI codes.
// ---------------------------------------------------------------------------

type lineBuffer struct {
	mu    sync.Mutex
	lines []string
}

func newLineBuffer() *lineBuffer {
	return &lineBuffer{lines: make([]string, 0, 128)}
}

func (lb *lineBuffer) drain(r io.Reader) {
	scanner := bufio.NewScanner(r)
	// Increase buffer size: the client might blast a lot of ANSI codes in one line.
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)
	for scanner.Scan() {
		lb.mu.Lock()
		lb.lines = append(lb.lines, scanner.Text())
		lb.mu.Unlock()
	}
}

// contains checks raw lines (including ANSI codes).
func (lb *lineBuffer) contains(substr string) bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for _, l := range lb.lines {
		if strings.Contains(l, substr) {
			return true
		}
	}
	return false
}

// containsClean checks lines after stripping ANSI escape codes.
func (lb *lineBuffer) containsClean(substr string) bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for _, l := range lb.lines {
		clean := stripANSI(l)
		if strings.Contains(clean, substr) {
			return true
		}
	}
	return false
}

func (lb *lineBuffer) snapshot() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return strings.Join(lb.lines, "\n")
}

// cleanSnapshot returns all captured output with ANSI codes stripped.
func (lb *lineBuffer) cleanSnapshot() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	cleaned := make([]string, len(lb.lines))
	for i, l := range lb.lines {
		cleaned[i] = stripANSI(l)
	}
	return strings.Join(cleaned, "\n")
}

func (lb *lineBuffer) clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.lines = lb.lines[:0]
}

// ---------------------------------------------------------------------------
// High-level client action helpers.
//
// Each helper:
//   1. Clears the output buffer (so we only see the new response).
//   2. Sends the menu selection.
//   3. Sends the field values in the exact order your C++ client expects.
//   4. Sends an extra Enter to satisfy wait_for_enter().
//   5. Returns the captured output for assertion.
//
// INPUT ORDER (from your C++ bankClient.cpp):
//   Open:     Name, Password, Currency(str), Balance
//   Close:    Name, AccNo, Password
//   Deposit:  Name, AccNo, Password, Currency(str), Amount
//   Withdraw: Name, AccNo, Password, Currency(str), Amount
//   Monitor:  Name, AccNo, Password   (then interval is asked separately)
//   Balance:  Name, AccNo, Password
//   Transfer: Name, AccNo, Password, ToAccNo, Amount
// ---------------------------------------------------------------------------

func openAccount(t *testing.T, cp *ClientProcess, name, password, currency, balance string) string {
	t.Helper()
	cp.clearOutput()
	cp.sendLines(MenuOpenAccount, name, password, currency, balance)

	// Wait for the server to respond with the account number.
	if !cp.waitForOutput("10", defaultTimeout) {
		// Might have printed a different format: grab whatever we have.
		time.Sleep(1 * time.Second)
	}

	// Press Enter to get past wait_for_enter() and back to the menu.
	cp.pressEnter()

	// Give the client time to reprint the menu.
	time.Sleep(300 * time.Millisecond)
	return cp.outputSnapshot()
}

func closeAccount(t *testing.T, cp *ClientProcess, name, accNo, password string) string {
	t.Helper()
	cp.clearOutput()
	cp.sendLines(MenuCloseAccount, name, accNo, password)
	time.Sleep(1 * time.Second)
	cp.pressEnter()
	time.Sleep(300 * time.Millisecond)
	return cp.outputSnapshot()
}

func deposit(t *testing.T, cp *ClientProcess, name, accNo, password, currency, amount string) string {
	t.Helper()
	cp.clearOutput()
	cp.sendLines(MenuDeposit, name, accNo, password, currency, amount)

	if !cp.waitForOutput(".", defaultTimeout) {
		time.Sleep(1 * time.Second)
	}
	cp.pressEnter()
	time.Sleep(300 * time.Millisecond)
	return cp.outputSnapshot()
}

func withdraw(t *testing.T, cp *ClientProcess, name, accNo, password, currency, amount string) string {
	t.Helper()
	cp.clearOutput()
	cp.sendLines(MenuWithdraw, name, accNo, password, currency, amount)

	if !cp.waitForOutput(".", defaultTimeout) {
		time.Sleep(1 * time.Second)
	}
	cp.pressEnter()
	time.Sleep(300 * time.Millisecond)
	return cp.outputSnapshot()
}

func getBalance(t *testing.T, cp *ClientProcess, name, accNo, password string) string {
	t.Helper()
	cp.clearOutput()
	cp.sendLines(MenuGetBalance, name, accNo, password)

	if !cp.waitForOutput(".", defaultTimeout) {
		time.Sleep(1 * time.Second)
	}
	cp.pressEnter()
	time.Sleep(300 * time.Millisecond)
	return cp.outputSnapshot()
}

func transfer(t *testing.T, cp *ClientProcess, name, fromAcc, password, toName, toAcc, currency, amount string) string {
	t.Helper()
	cp.clearOutput()
	cp.sendLines(MenuTransfer, name, fromAcc, password, toName, toAcc, currency, amount)

	if !cp.waitForOutput(".", defaultTimeout) {
		time.Sleep(1 * time.Second)
	}
	cp.pressEnter()
	time.Sleep(300 * time.Millisecond)
	return cp.outputSnapshot()
}

func registerMonitor(t *testing.T, cp *ClientProcess, name, accNo, password, intervalSec string) {
	t.Helper()
	cp.clearOutput()
	// Monitor calls fill_auth_details first (name, accNo, pw), then asks for interval.
	// But based on the checklist, monitor just needs the interval.
	// Your actual implementation may vary: adjust these fields if needed.
	cp.sendLines(MenuMonitor, name, accNo, password, intervalSec)
	// Monitor blocks the client, so we do NOT press Enter: it's waiting for callbacks.
}

func exitClient(cp *ClientProcess) {
	cp.send(MenuExit)
	time.Sleep(500 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Assertion and extraction helpers.
// ---------------------------------------------------------------------------

func assertContains(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("[%s] expected output to contain %q, but it didn't.\nFull output:\n%s",
			context, needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("[%s] expected output NOT to contain %q, but it did.\nFull output:\n%s",
			context, needle, haystack)
	}
}

func assertContainsAnyOf(t *testing.T, haystack string, needles []string, context string) {
	t.Helper()
	lower := strings.ToLower(haystack)
	for _, n := range needles {
		if strings.Contains(lower, strings.ToLower(n)) {
			return
		}
	}
	t.Errorf("[%s] expected output to contain one of %v, but it didn't.\nFull output:\n%s",
		context, needles, haystack)
}

// extractAccountNumber pulls the first number >= 10000 from the output string.
func extractAccountNumber(output string) (string, bool) {
	// Match any number >= 10000 that appears in the output,
	// even if glued to non-digit characters like box-drawing chars.
	re := regexp.MustCompile(`\b(\d{5,})\b`)
	// \b won't match against unicode box chars, so also try a looser pattern:
	if matches := re.FindAllString(output, -1); len(matches) > 0 {
		for _, m := range matches {
			n, err := strconv.Atoi(m)
			if err == nil && n >= 10000 {
				return m, true
			}
		}
	}

	// Fallback: scan for digit sequences anywhere in the string.
	re2 := regexp.MustCompile(`(\d{5,})`)
	for _, m := range re2.FindAllString(output, -1) {
		n, err := strconv.Atoi(m)
		if err == nil && n >= 10000 {
			return strconv.Itoa(n), true
		}
	}

	return "", false
}

func extractBalance(output string) (float64, bool) {
	// The C++ client prints "Balance        : <value>" in its response box.
	// We use a regex to grab the number directly after "Balance" + any
	// whitespace/colon combo, which avoids confusion with menu items like
	// "6. BALANCE" that don't have a colon followed by a number.
	re := regexp.MustCompile(`(?i)balance\s*:\s*([0-9]+\.?[0-9]*)`)
	if m := re.FindStringSubmatch(output); len(m) == 2 {
		f, err := strconv.ParseFloat(m[1], 64)
		if err == nil {
			return f, true
		}
	}

	// Fallback: return the last non-negative float in the output.
	var lastVal float64
	found := false
	for _, word := range strings.Fields(output) {
		word = strings.TrimRight(word, ".,;:!?()[]")
		f, err := strconv.ParseFloat(word, 64)
		if err == nil && f >= 0 {
			lastVal = f
			found = true
		}
	}
	return lastVal, found
}

func floatEquals(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

// ---------------------------------------------------------------------------
// lockServer grabs the global mutex so only one test uses port 2222 at a time.
// It returns an unlock function that should be deferred.
// ---------------------------------------------------------------------------
func lockServer(t *testing.T) {
	t.Helper()
	serverMu.Lock()
	t.Cleanup(func() { serverMu.Unlock() })
}

// ==========================================================================
//
//  SECTION 2A: OPEN ACCOUNT
//
// ==========================================================================

func TestOpenAccount_HappyPath(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")

	accNo, ok := extractAccountNumber(out)
	if !ok {
		t.Fatalf("could not find account number (>= 10000) in output:\n%s", out)
	}
	t.Logf("Account opened with number: %s", accNo)

	exitClient(client)
}

func TestOpenAccount_UniqueIDs(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out1 := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	acc1, ok1 := extractAccountNumber(out1)

	out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "500.00")
	acc2, ok2 := extractAccountNumber(out2)

	out3 := openAccount(t, client, "Charlie", "pass9999", CurrSGD, "250.00")
	acc3, ok3 := extractAccountNumber(out3)

	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("failed to extract account numbers: got %q, %q, %q", acc1, acc2, acc3)
	}

	if acc1 == acc2 || acc2 == acc3 || acc1 == acc3 {
		t.Errorf("account numbers are not unique: %s, %s, %s", acc1, acc2, acc3)
	}
	t.Logf("Unique IDs: %s, %s, %s", acc1, acc2, acc3)

	exitClient(client)
}

func TestOpenAccount_DifferentCurrencies(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	for _, curr := range []string{CurrSGD, CurrUSD, CurrEUR} {
		out := openAccount(t, client, "TestUser", "pass1234", curr, "100.00")
		_, ok := extractAccountNumber(out)
		if !ok {
			t.Errorf("failed to open account with currency %s:\n%s", curr, out)
		}
	}

	exitClient(client)
}

// ==========================================================================
//
//  SECTION 2B: CLOSE ACCOUNT
//
// ==========================================================================

func TestCloseAccount_HappyPath(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	closeOut := closeAccount(t, client, "Alice", accNo, "pass1234")
	assertNotContains(t, strings.ToLower(closeOut), "error", "close happy path")

	// Verify account is gone.
	balOut := getBalance(t, client, "Alice", accNo, "pass1234")
	assertContainsAnyOf(t, balOut, []string{"error", "not found", "invalid"}, "balance after close should fail")

	exitClient(client)
}

func TestCloseAccount_WrongPassword(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	closeOut := closeAccount(t, client, "Alice", accNo, "wrongpwd")
	assertContainsAnyOf(t, closeOut, []string{"error", "invalid"}, "close wrong password")

	// Account must still exist.
	balOut := getBalance(t, client, "Alice", accNo, "pass1234")
	assertContains(t, balOut, "1000", "balance after failed close")

	exitClient(client)
}

func TestCloseAccount_NameMismatch(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	closeOut := closeAccount(t, client, "Bob", accNo, "pass1234")
	assertContainsAnyOf(t, closeOut, []string{"error", "mismatch", "invalid"}, "close name mismatch")

	exitClient(client)
}

func TestCloseAccount_NonExistent(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	closeOut := closeAccount(t, client, "Alice", "99999", "pass1234")
	assertContainsAnyOf(t, closeOut, []string{"error", "not found", "invalid"}, "close non-existent")

	exitClient(client)
}

// ==========================================================================
//
//  SECTION 2C: DEPOSIT & WITHDRAW
//
// ==========================================================================

func TestDeposit_HappyPath(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	depOut := deposit(t, client, "Alice", accNo, "pass1234", CurrSGD, "500.00")
	assertContains(t, depOut, "1500", "deposit new balance")

	exitClient(client)
}

func TestWithdraw_HappyPath(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	wdOut := withdraw(t, client, "Alice", accNo, "pass1234", CurrSGD, "300.00")
	assertContains(t, wdOut, "700", "withdraw new balance")

	exitClient(client)
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "100.00")
	accNo, _ := extractAccountNumber(out)

	wdOut := withdraw(t, client, "Alice", accNo, "pass1234", CurrSGD, "500.00")
	assertContainsAnyOf(t, wdOut, []string{"error", "insufficient"}, "withdraw insufficient funds")

	balOut := getBalance(t, client, "Alice", accNo, "pass1234")
	assertContains(t, balOut, "100", "balance unchanged after failed withdraw")

	exitClient(client)
}

func TestDepositWithdraw_WrongPassword(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	depOut := deposit(t, client, "Alice", accNo, "wrongpwd", CurrSGD, "500.00")
	assertContainsAnyOf(t, depOut, []string{"error", "invalid"}, "deposit wrong password")

	wdOut := withdraw(t, client, "Alice", accNo, "wrongpwd", CurrSGD, "100.00")
	assertContainsAnyOf(t, wdOut, []string{"error", "invalid"}, "withdraw wrong password")

	exitClient(client)
}

func TestDepositWithdraw_CurrencyMismatch(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	depOut := deposit(t, client, "Alice", accNo, "pass1234", CurrUSD, "500.00")
	assertContainsAnyOf(t, depOut, []string{"error", "mismatch", "currency"}, "deposit currency mismatch")

	wdOut := withdraw(t, client, "Alice", accNo, "pass1234", CurrEUR, "100.00")
	assertContainsAnyOf(t, wdOut, []string{"error", "mismatch", "currency"}, "withdraw currency mismatch")

	exitClient(client)
}

func TestDeposit_FloatingPointPrecision(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "0.00")
	accNo, _ := extractAccountNumber(out)

	for i := 0; i < 3; i++ {
		deposit(t, client, "Alice", accNo, "pass1234", CurrSGD, "0.10")
	}

	balOut := getBalance(t, client, "Alice", accNo, "pass1234")
	assertContains(t, balOut, "0.3", "floating point precision after 3 deposits of 0.10")
	bal, ok := extractBalance(balOut)
	if !ok {
		t.Fatalf("could not extract balance:\n%s", balOut)
	}

	if !floatEquals(bal, 0.30, 0.02) {
		t.Errorf("expected balance ~0.30, got %.4f", bal)
	}

	exitClient(client)
}

// ==========================================================================
//
//  SECTION 2D: MONITOR (CALLBACK)
//
// ==========================================================================

func TestMonitor_ReceivesOpenAccountUpdate(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")

	// Client B registers for monitoring.
	clientB := startClient(t, SemanticsAtLeastOnce)
	registerMonitor(t, clientB, "Monitor", "10000", "pass1234", "30")
	time.Sleep(1 * time.Second)

	// Client A opens an account.
	clientA := startClient(t, SemanticsAtLeastOnce)
	openAccount(t, clientA, "Dave", "pass1234", CurrSGD, "500.00")

	// Client B should receive a callback.
	if !clientB.waitForOutput("Dave", 5*time.Second) && !clientB.waitForOutput("500", 3*time.Second) {
		t.Logf("Monitor output:\n%s", clientB.outputSnapshot())
		t.Error("monitoring client did not receive OpenAccount callback")
	}

	exitClient(clientA)
}

func TestMonitor_ReceivesDepositUpdate(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")

	clientA := startClient(t, SemanticsAtLeastOnce)
	out := openAccount(t, clientA, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	clientB := startClient(t, SemanticsAtLeastOnce)
	registerMonitor(t, clientB, "Monitor", "10000", "pass1234", "30")
	time.Sleep(1 * time.Second)

	deposit(t, clientA, "Alice", accNo, "pass1234", CurrSGD, "200.00")

	if !clientB.waitForOutput("1200", 5*time.Second) {
		t.Logf("Monitor output:\n%s", clientB.outputSnapshot())
		t.Error("monitoring client did not receive deposit callback")
	}

	exitClient(clientA)
}

func TestMonitor_ReceivesWithdrawUpdate(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")

	clientA := startClient(t, SemanticsAtLeastOnce)
	out := openAccount(t, clientA, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	clientB := startClient(t, SemanticsAtLeastOnce)
	registerMonitor(t, clientB, "Monitor", "10000", "pass1234", "30")
	time.Sleep(1 * time.Second)
	clientB.clearOutput()

	withdraw(t, clientA, "Alice", accNo, "pass1234", CurrSGD, "300.00")

	if !clientB.waitForOutput("700", 5*time.Second) {
		t.Logf("Monitor output:\n%s", clientB.outputSnapshot())
		t.Error("monitoring client did not receive withdraw callback")
	}

	exitClient(clientA)
}

func TestMonitor_NoCallbackOnFailedOperation(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")

	clientSetup := startClient(t, SemanticsAtLeastOnce)
	out := openAccount(t, clientSetup, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	clientB := startClient(t, SemanticsAtLeastOnce)
	registerMonitor(t, clientB, "Monitor", "10000", "pass1234", "30")
	time.Sleep(1 * time.Second)
	clientB.clearOutput()

	// Failed withdraw (wrong password): should NOT trigger callback.
	clientA := startClient(t, SemanticsAtLeastOnce)
	withdraw(t, clientA, "Alice", accNo, "wrongpwd", CurrSGD, "100.00")

	time.Sleep(2 * time.Second)
	snapshot := clientB.outputSnapshot()
	if strings.Contains(snapshot, "900") {
		t.Errorf("monitor received callback for failed operation:\n%s", snapshot)
	}

	exitClient(clientA)
	exitClient(clientSetup)
}

func TestMonitor_MultipleConcurrentMonitors(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")

	clientB := startClient(t, SemanticsAtLeastOnce)
	registerMonitor(t, clientB, "Monitor", "10000", "pass1234", "30")
	time.Sleep(500 * time.Millisecond)

	clientC := startClient(t, SemanticsAtLeastOnce)
	registerMonitor(t, clientC, "Monitor", "10000", "pass1234", "30")
	time.Sleep(500 * time.Millisecond)

	clientB.clearOutput()
	clientC.clearOutput()

	clientA := startClient(t, SemanticsAtLeastOnce)
	openAccount(t, clientA, "Shared", "pass1234", CurrSGD, "999.00")

	bGot := clientB.waitForOutput("Shared", 5*time.Second) || clientB.waitForOutput("999", 5*time.Second)
	cGot := clientC.waitForOutput("Shared", 5*time.Second) || clientC.waitForOutput("999", 5*time.Second)

	if !bGot {
		t.Errorf("Client B missed callback. Output:\n%s", clientB.outputSnapshot())
	}
	if !cGot {
		t.Errorf("Client C missed callback. Output:\n%s", clientC.outputSnapshot())
	}

	exitClient(clientA)
}

// ==========================================================================
//
//  SECTION 3: CUSTOM OPERATIONS (GetBalance & Transfer)
//
// ==========================================================================

func TestGetBalance_HappyPath(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	balOut := getBalance(t, client, "Alice", accNo, "pass1234")
	assertContains(t, balOut, "1000", "getBalance happy path")

	exitClient(client)
}

func TestGetBalance_Idempotency(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	for i := 0; i < 5; i++ {
		balOut := getBalance(t, client, "Alice", accNo, "pass1234")
		assertContains(t, balOut, "1000", fmt.Sprintf("idempotency check #%d", i+1))
	}

	exitClient(client)
}

func TestGetBalance_WrongCredentials(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	balOut := getBalance(t, client, "Alice", accNo, "wrongpwd")
	assertContainsAnyOf(t, balOut, []string{"error", "invalid"}, "getBalance wrong password")

	balOut2 := getBalance(t, client, "NotAlice", accNo, "pass1234")
	assertContainsAnyOf(t, balOut2, []string{"error", "mismatch", "invalid"}, "getBalance wrong name")

	exitClient(client)
}

func TestTransfer_HappyPath(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out1 := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accAlice, _ := extractAccountNumber(out1)

	out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "500.00")
	accBob, _ := extractAccountNumber(out2)

	txOut := transfer(t, client, "Alice", accAlice, "pass1234", "Bob", accBob, CurrSGD, "200.00")
	assertContains(t, txOut, "800", "transfer: Alice's new balance")

	bobBal := getBalance(t, client, "Bob", accBob, "pass5678")
	assertContains(t, bobBal, "700", "transfer: Bob's new balance")

	exitClient(client)
}

func TestTransfer_NonIdempotencyProof(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out1 := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accAlice, _ := extractAccountNumber(out1)
	out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "500.00")
	accBob, _ := extractAccountNumber(out2)

	transfer(t, client, "Alice", accAlice, "pass1234", "Bob", accBob, CurrSGD, "100.00")
	transfer(t, client, "Alice", accAlice, "pass1234", "Bob", accBob, CurrSGD, "100.00")

	aliceBal := getBalance(t, client, "Alice", accAlice, "pass1234")
	assertContains(t, aliceBal, "800", "non-idempotent: Alice after 2 transfers")

	bobBal := getBalance(t, client, "Bob", accBob, "pass5678")
	assertContains(t, bobBal, "700", "non-idempotent: Bob after 2 transfers")

	exitClient(client)
}

func TestTransfer_InsufficientFunds(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out1 := openAccount(t, client, "Alice", "pass1234", CurrSGD, "100.00")
	accAlice, _ := extractAccountNumber(out1)
	out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "500.00")
	accBob, _ := extractAccountNumber(out2)

	txOut := transfer(t, client, "Alice", accAlice, "pass1234", "Bob", accBob, CurrSGD, "500.00")
	assertContainsAnyOf(t, txOut, []string{"error", "insufficient"}, "transfer insufficient funds")

	exitClient(client)
}

func TestTransfer_SameAccount(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accAlice, _ := extractAccountNumber(out)

	txOut := transfer(t, client, "Alice", accAlice, "pass1234", "Alice", accAlice, CurrSGD, "100.00")
	assertContainsAnyOf(t, txOut, []string{"error", "same"}, "transfer to self")

	exitClient(client)
}

func TestTransfer_DestinationNotFound(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accAlice, _ := extractAccountNumber(out)

	txOut := transfer(t, client, "Alice", accAlice, "pass1234", "Bob", "99999", CurrSGD, "100.00")
	assertContainsAnyOf(t, txOut, []string{"error", "not found", "not_found", "invalid"}, "transfer dest not found")

	aliceBal := getBalance(t, client, "Alice", accAlice, "pass1234")
	assertContains(t, aliceBal, "1000", "alice balance unchanged")

	exitClient(client)
}

// ==========================================================================
//
//  SECTION 4: FAULT TOLERANCE & INVOCATION SEMANTICS
//
// ==========================================================================

func TestCLIToggle_AtLeastOnce(t *testing.T) {
	lockServer(t)
	srv := startServer(t, "at-least-once")

	if !srv.waitForOutput("at-least-once", 3*time.Second) {
		t.Logf("Server output:\n%s", srv.allOutput())
		t.Error("server did not log at-least-once mode")
	}

	client := startClient(t, SemanticsAtLeastOnce)
	out := openAccount(t, client, "Test", "testpass", CurrSGD, "100.00")
	if _, ok := extractAccountNumber(out); !ok {
		t.Error("could not open account in at-least-once mode")
	}
	exitClient(client)
}

func TestCLIToggle_AtMostOnce(t *testing.T) {
	lockServer(t)
	srv := startServer(t, "at-most-once")

	if !srv.waitForOutput("at-most-once", 3*time.Second) {
		t.Logf("Server output:\n%s", srv.allOutput())
		t.Error("server did not log at-most-once mode")
	}

	client := startClient(t, SemanticsAtMostOnce)
	out := openAccount(t, client, "Test", "testpass", CurrSGD, "100.00")
	if _, ok := extractAccountNumber(out); !ok {
		t.Error("could not open account in at-most-once mode")
	}
	exitClient(client)
}

func TestAtMostOnce_DepositSafeUnderLoss(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-most-once", "0.5")
	client := startClient(t, SemanticsAtMostOnce)

	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accNo, _ := extractAccountNumber(out)

	deposit(t, client, "Alice", accNo, "pass1234", CurrSGD, "500.00")
	time.Sleep(3 * time.Second)

	balOut := getBalance(t, client, "Alice", accNo, "pass1234")
	bal, ok := extractBalance(balOut)
	if ok && bal > 1600 {
		t.Errorf("at-most-once FAILED: deposit duplicated. Balance=%.2f (expected 1500)", bal)
	}
	if ok {
		t.Logf("at-most-once deposit: balance=%.2f", bal)
	}

	exitClient(client)
}

func TestAtMostOnce_TransferSafeUnderLoss(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-most-once", "0.5")
	client := startClient(t, SemanticsAtMostOnce)

	out1 := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accAlice, _ := extractAccountNumber(out1)
	out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "500.00")
	accBob, _ := extractAccountNumber(out2)

	transfer(t, client, "Alice", accAlice, "pass1234", "Bob", accBob, CurrSGD, "100.00")
	time.Sleep(3 * time.Second)

	aliceBal := getBalance(t, client, "Alice", accAlice, "pass1234")
	aliceVal, _ := extractBalance(aliceBal)
	bobBal := getBalance(t, client, "Bob", accBob, "pass5678")
	bobVal, _ := extractBalance(bobBal)

	t.Logf("at-most-once transfer: Alice=%.2f, Bob=%.2f", aliceVal, bobVal)

	if aliceVal < 850 {
		t.Errorf("at-most-once FAILED: transfer duplicated. Alice=%.2f (expected ~900)", aliceVal)
	}

	exitClient(client)
}

func TestAtLeastOnce_TransferBreaksUnderLoss(t *testing.T) {
	lockServer(t)
	srv := startServer(t, "at-least-once", "0.3")
	client := startClient(t, SemanticsAtLeastOnce)

	out1 := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accAlice, _ := extractAccountNumber(out1)
	out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "500.00")
	accBob, _ := extractAccountNumber(out2)

	transfer(t, client, "Alice", accAlice, "pass1234", "Bob", accBob, CurrSGD, "100.00")
	time.Sleep(3 * time.Second)

	aliceBal := getBalance(t, client, "Alice", accAlice, "pass1234")
	aliceVal, _ := extractBalance(aliceBal)
	bobBal := getBalance(t, client, "Bob", accBob, "pass5678")
	bobVal, _ := extractBalance(bobBal)

	t.Logf("[at-least-once] Alice=%.2f, Bob=%.2f (correct: 900, 600)", aliceVal, bobVal)

	if aliceVal < 850 {
		t.Logf("REPORT EVIDENCE: Transfer duplicated! At-least-once broke non-idempotent op.")
	}
	_ = srv

	exitClient(client)
}

func TestComparison_SameScenarioBothModes(t *testing.T) {
	for _, mode := range []string{"at-least-once", "at-most-once"} {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			lockServer(t)
			srv := startServer(t, mode, "0.3")
			clientFlag := SemanticsAtLeastOnce
			if mode == "at-most-once" {
				clientFlag = SemanticsAtMostOnce
			}
			client := startClient(t, clientFlag)

			out1 := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
			accAlice, _ := extractAccountNumber(out1)
			out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "500.00")
			accBob, _ := extractAccountNumber(out2)

			transfer(t, client, "Alice", accAlice, "pass1234", "Bob", accBob, CurrSGD, "100.00")
			time.Sleep(3 * time.Second)

			aliceBal := getBalance(t, client, "Alice", accAlice, "pass1234")
			aliceVal, _ := extractBalance(aliceBal)
			bobBal := getBalance(t, client, "Bob", accBob, "pass5678")
			bobVal, _ := extractBalance(bobBal)

			t.Logf("[%s] Alice=%.2f, Bob=%.2f", mode, aliceVal, bobVal)

			if mode == "at-most-once" && srv.outputContains("duplicate") {
				t.Logf("[%s] Server filtered duplicates", mode)
			}
			if mode == "at-most-once" && aliceVal < 850 {
				t.Errorf("[at-most-once] Alice=%.2f: duplicate filter may have failed", aliceVal)
			}

			exitClient(client)
		})
	}
}

// ==========================================================================
//
//  SECTION 1: PERSISTENCE
//
// ==========================================================================

func TestPersistence_AcrossMultipleRequests(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	out1 := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	acc1, _ := extractAccountNumber(out1)
	out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "2000.00")
	acc2, _ := extractAccountNumber(out2)
	out3 := openAccount(t, client, "Charlie", "pass9999", CurrSGD, "3000.00")
	acc3, _ := extractAccountNumber(out3)

	assertContains(t, getBalance(t, client, "Alice", acc1, "pass1234"), "1000", "persistence Alice")
	assertContains(t, getBalance(t, client, "Bob", acc2, "pass5678"), "2000", "persistence Bob")
	assertContains(t, getBalance(t, client, "Charlie", acc3, "pass9999"), "3000", "persistence Charlie")

	deposit(t, client, "Alice", acc1, "pass1234", CurrSGD, "500.00")
	withdraw(t, client, "Bob", acc2, "pass5678", CurrSGD, "300.00")

	assertContains(t, getBalance(t, client, "Alice", acc1, "pass1234"), "1500", "persistence Alice after deposit")
	assertContains(t, getBalance(t, client, "Bob", acc2, "pass5678"), "1700", "persistence Bob after withdraw")
	assertContains(t, getBalance(t, client, "Charlie", acc3, "pass9999"), "3000", "persistence Charlie unchanged")

	exitClient(client)
}

// ==========================================================================
//
//  SECTION 5: CLIENT INTERFACE & SERVER OUTPUT
//
// ==========================================================================

func TestClientMenu_ShowsAllOptions(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	time.Sleep(500 * time.Millisecond)
	out := client.outputSnapshot()
	lower := strings.ToLower(out)

	for _, kw := range []string{"open", "close", "deposit", "withdraw", "monitor", "balance", "transfer"} {
		if !strings.Contains(lower, kw) {
			t.Errorf("menu missing keyword %q.\nMenu output:\n%s", kw, out)
		}
	}

	exitClient(client)
}

func TestClientExit_CleansUp(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	exitClient(client)

	done := make(chan error, 1)
	go func() { done <- client.cmd.Wait() }()

	select {
	case <-done:
		// Process exited: success.
	case <-time.After(5 * time.Second):
		t.Error("client did not exit within 5 seconds")
	}
}

func TestServerPrints_IncomingRequests(t *testing.T) {
	lockServer(t)
	srv := startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	openAccount(t, client, "Alice", "pass1234", CurrSGD, "100.00")

	if !srv.waitForOutput("Received", 3*time.Second) && !srv.waitForOutput("request", 3*time.Second) {
		t.Logf("Server output:\n%s", srv.allOutput())
		t.Error("server did not log incoming request")
	}

	exitClient(client)
}

// ==========================================================================
//
//  SECTION 6: CROSS-LANGUAGE INTEROPERABILITY
//
// ==========================================================================

func TestCrossLanguage_AllServicesWork(t *testing.T) {
	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	// S1: Open
	out := openAccount(t, client, "Alice", "pass1234", CurrSGD, "1000.00")
	accAlice, ok := extractAccountNumber(out)
	if !ok {
		t.Fatal("S1 Open failed")
	}
	out2 := openAccount(t, client, "Bob", "pass5678", CurrSGD, "500.00")
	accBob, ok := extractAccountNumber(out2)
	if !ok {
		t.Fatal("S1 Open (Bob) failed")
	}

	// S3: Deposit
	depOut := deposit(t, client, "Alice", accAlice, "pass1234", CurrSGD, "250.00")
	assertContains(t, depOut, "1250", "S3 Deposit")

	// S3: Withdraw
	wdOut := withdraw(t, client, "Alice", accAlice, "pass1234", CurrSGD, "100.00")
	assertContains(t, wdOut, "1150", "S3 Withdraw")

	// S6: GetBalance
	balOut := getBalance(t, client, "Alice", accAlice, "pass1234")
	assertContains(t, balOut, "1150", "S6 GetBalance")

	// S7: Transfer
	txOut := transfer(t, client, "Alice", accAlice, "pass1234", "Bob", accBob, CurrSGD, "150.00")
	assertContains(t, txOut, "1000", "S7 Transfer Alice")

	bobBal := getBalance(t, client, "Bob", accBob, "pass5678")
	assertContains(t, bobBal, "650", "S7 Transfer Bob")

	// S2: Close
	closeOut := closeAccount(t, client, "Bob", accBob, "pass5678")
	assertNotContains(t, strings.ToLower(closeOut), "error", "S2 Close")

	deadBal := getBalance(t, client, "Bob", accBob, "pass5678")
	assertContainsAnyOf(t, deadBal, []string{"error", "not found", "invalid"}, "S2 verify gone")

	t.Log("All 7 services passed cross-language interop!")
	exitClient(client)
}

// ==========================================================================
//
//  BONUS: STRESS TEST
//
// ==========================================================================

func TestStress_RapidOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	lockServer(t)
	_ = startServer(t, "at-least-once")
	client := startClient(t, SemanticsAtLeastOnce)

	accounts := make([]string, 5)
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("Stress%d", i)
		out := openAccount(t, client, name, "stresspass", CurrSGD, "1000.00")
		accNo, ok := extractAccountNumber(out)
		if !ok {
			t.Fatalf("failed to open stress account %d", i)
		}
		accounts[i] = accNo
	}

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("Stress%d", i%5)
		deposit(t, client, name, accounts[i%5], "stresspass", CurrSGD, "10.00")
	}

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("Stress%d", i)
		balOut := getBalance(t, client, name, accounts[i], "stresspass")
		if strings.Contains(strings.ToLower(balOut), "error") {
			t.Errorf("account %s inaccessible after stress test", accounts[i])
		}
	}

	t.Log("Stress test completed: no crashes")
	exitClient(client)
}
