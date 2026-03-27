# Integration Test Suite — Distributed Banking System

## Overview

This test suite automates the manual testing checklist (`testing_checklist_1_.md`) by driving the **real C++ client binary** against the **real Go server binary** on localhost. Tests pipe commands through stdin and assert on stdout output.

## Checklist Coverage

| Checklist # | Test Name                                   | Status       |
|-------------|---------------------------------------------|--------------|
| 1.9         | `TestPersistence_AcrossMultipleRequests`     | ✅ Automated  |
| 2.1         | `TestOpenAccount_HappyPath`                 | ✅ Automated  |
| 2.2         | `TestOpenAccount_UniqueIDs`                 | ✅ Automated  |
| 2.3         | `TestOpenAccount_DifferentCurrencies`       | ✅ Automated  |
| 2.4         | `TestCloseAccount_HappyPath`               | ✅ Automated  |
| 2.5         | `TestCloseAccount_WrongPassword`            | ✅ Automated  |
| 2.6         | `TestCloseAccount_NameMismatch`             | ✅ Automated  |
| 2.7         | `TestCloseAccount_NonExistent`              | ✅ Automated  |
| 2.8         | `TestDeposit_HappyPath`                    | ✅ Automated  |
| 2.9         | `TestWithdraw_HappyPath`                   | ✅ Automated  |
| 2.10        | `TestWithdraw_InsufficientFunds`            | ✅ Automated  |
| 2.11        | `TestDepositWithdraw_WrongPassword`         | ✅ Automated  |
| 2.12        | `TestDepositWithdraw_CurrencyMismatch`      | ✅ Automated  |
| 2.13        | `TestDeposit_FloatingPointPrecision`        | ✅ Automated  |
| 2.14        | `TestMonitor_Registration_HappyPath`        | ✅ Automated  |
| 2.15        | `TestMonitor_ReceivesOpenAccountUpdate`     | ✅ Automated  |
| 2.16        | `TestMonitor_ReceivesDepositUpdate`         | ✅ Automated  |
| 2.17        | `TestMonitor_ReceivesWithdrawUpdate`        | ✅ Automated  |
| 2.18        | `TestMonitor_ReceivesCloseAccountUpdate`    | ✅ Automated  |
| 2.20        | `TestMonitor_ExpirationAndCleanup`          | ✅ Automated  |
| 2.21        | `TestMonitor_MultipleConcurrentMonitors`    | ✅ Automated  |
| 2.22        | `TestMonitor_NoCallbackOnFailedOperation`   | ✅ Automated  |
| 3.1         | `TestGetBalance_HappyPath`                 | ✅ Automated  |
| 3.2         | `TestGetBalance_Idempotency`                | ✅ Automated  |
| 3.3         | `TestGetBalance_WrongCredentials`           | ✅ Automated  |
| 3.4         | `TestTransfer_HappyPath`                   | ✅ Automated  |
| 3.5         | `TestTransfer_NonIdempotencyProof`          | ✅ Automated  |
| 3.6         | `TestTransfer_InsufficientFunds`            | ✅ Automated  |
| 3.7         | `TestTransfer_SameAccount`                  | ✅ Automated  |
| 3.8         | `TestTransfer_DestinationNotFound`          | ✅ Automated  |
| 3.9         | `TestTransfer_TriggersMonitorCallback`      | ✅ Automated  |
| 4.1         | `TestCLIToggle_AtLeastOnce`                 | ✅ Automated  |
| 4.2         | `TestCLIToggle_AtMostOnce`                  | ✅ Automated  |
| 4.3         | `TestLossSimulator_DropsPackets`            | ✅ Automated  |
| 4.5         | `TestAtLeastOnce_GetBalanceIdempotentSafe`  | ✅ Automated  |
| 4.6         | `TestAtLeastOnce_TransferBreaksUnderLoss`   | ✅ Automated  |
| 4.7         | `TestAtLeastOnce_DepositDuplicatedUnderLoss`| ✅ Automated  |
| 4.8         | `TestAtMostOnce_DuplicateFilteringServerLog`| ✅ Automated  |
| 4.9         | `TestAtMostOnce_ReplyHistoryCache`          | ✅ Automated  |
| 4.10        | `TestAtMostOnce_DepositSafeUnderLoss`       | ✅ Automated  |
| 4.11        | `TestAtMostOnce_TransferSafeUnderLoss`      | ✅ Automated  |
| 4.12        | `TestRequestIDs_MonotonicallyIncreasing`    | ✅ Automated  |
| 4.13        | `TestComparison_SameScenarioBothModes`      | ✅ Automated  |
| 5.1         | `TestClientMenu_ShowsAllOptions`            | ✅ Automated  |
| 5.2         | `TestClientExit_CleansUp`                  | ✅ Automated  |
| 5.3         | `TestClient_PrintsServerResponses`          | ✅ Automated  |
| 5.4         | `TestServerPrints_IncomingRequests`         | ✅ Automated  |
| 6.4         | `TestCrossLanguage_AllServicesWork`         | ✅ Automated  |
| 7.1         | `TestWireFormat_RequestMinimumSize`         | ✅ Partial    |
| —           | `TestStress_RapidOperations`                | ✅ Bonus      |

### Not Automated (require physical multi-machine setup or manual hex inspection)

| Checklist # | Reason                                       |
|-------------|----------------------------------------------|
| 1.1–1.4     | Network/socket verification (netstat, manual) |
| 1.5–1.8     | Hex-dump byte-level inspection (manual)       |
| 2.19        | Monitor client-side blocking (no threading)   |
| 4.4         | Precise retransmission timing observation     |
| 6.1–6.3     | Live demo with multiple laptops               |
| 7.2–7.3     | Reply/callback hex-dump verification          |

## Prerequisites

1. **Build the Go server binary:**
   ```bash
   cd bank-server/
   go build -o ./bin/server ./cmd/main.go
   ```

2. **Build the C++ client binary:**
   ```bash
   cd bank-client/
   make   # or cmake --build build/
   # Output should be at ./bin/client (or wherever your Makefile puts it)
   ```

3. **Verify both binaries exist** at the expected paths, or set env vars.

## Running

### Basic run
```bash
cd integration_test/
go test -v -timeout 120s .
```

### Custom binary paths
```bash
SERVER_BIN=/path/to/server CLIENT_BIN=/path/to/client go test -v -timeout 120s .
```

### Run a specific test
```bash
go test -v -run TestTransfer_HappyPath -timeout 30s .
```

### Run a category of tests
```bash
go test -v -run "TestOpenAccount" -timeout 30s .    # All open account tests
go test -v -run "TestTransfer" -timeout 30s .        # All transfer tests
go test -v -run "TestAtMostOnce" -timeout 60s .      # All at-most-once tests
go test -v -run "TestMonitor" -timeout 60s .         # All monitor tests
go test -v -run "TestComparison" -timeout 120s .     # The side-by-side experiment
```

## Configuration

### Timing

If you're running on a slow machine or over a real network, you may need to increase:
- `defaultTimeout` (wait for client responses)
- `time.Sleep` calls in `startServer` and `startClient` (wait for process startup)

## Troubleshooting

**"could not find account number in output"**
→ Your client's output format for account numbers might be different. Check `extractAccountNumber()` and adjust the parsing logic.

**"timed out waiting for..."**
→ Increase `defaultTimeout` or the specific sleep in the helper function. Also check that the server actually started by looking at its stderr output in the test logs.

**"server did not log at-least-once mode"**
→ Make sure your server prints the mode string to stdout or stderr on startup. The test looks for the substring "at-least-once" in the combined output.

**Monitor tests failing**
→ Monitor tests launch two separate client processes. Make sure your client binary doesn't conflict when multiple instances run simultaneously on the same machine.