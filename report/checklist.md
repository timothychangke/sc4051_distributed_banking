# PROJECT CHECKLIST VERIFICATION REPORT

_SC4051 Distributed Banking System: Generated 2026-03-27_

---

## 1. INFRASTRUCTURE & NETWORKING

| Item                          | Status   | Evidence                                                                                                                                                                                  |
| ----------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| UDP ONLY                      | **PASS** | Client: `SOCK_DGRAM` in `udpSocket.cpp:7`. Server: `net.ListenUDP("udp", ...)` in `main.go:66`. No TCP anywhere.                                                                          |
| CUSTOM MARSHALLING            | **PASS** | Client: field-by-field TLV encoding in `cmdEncoder.cpp` / `msgSerializer.cpp`. Server: manual `binary.BigEndian` reads in `marshalling/`. No protobuf, boost, or JSON libraries.          |
| BYTE ORDER                    | **PASS** | Client: `htonl`/`htons`/`ntohl`/`ntohs` on every multi-byte field in `msgSerializer.cpp` and `cmdEncoder.cpp`. Server: `binary.BigEndian.PutUint16/32/64` throughout `encoder.go`.        |
| STRING HANDLING               | **PASS** | Client: `append_uint32(length)` then `append_string(data)` in `cmdEncoder.cpp:247`. Server: `PutLengthPrefixedString()` in `encoder.go:84`. Both sides use `[4-byte BE length][N bytes]`. |
| SERVER ADDRESSING             | **PASS** | `client/src/main.cpp:13-19`:requires `argv[1]` (IP) and `argv[2]` (port); prints usage and exits if missing.                                                                              |
| LAB COMPATIBILITY (Port 2222) | **PASS** | `server/cmd/main.go:36`:`defaultPort = 2222`.                                                                                                                                             |
| PERSISTENCE (in-memory)       | **PASS** | `server/internal/store/memory.go:12-25`:`accounts map[uint32]*models.Account`, no disk I/O anywhere.                                                                                      |

**Section result: 7 / 7 PASS**

---

## 2. CORE BANKING SERVICES (S1-S4)

| Item                       | Status   | Evidence                                                                                                                                                                          |
| -------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| S1 - OPEN                  | **PASS** | Already marked complete.                                                                                                                                                          |
| S2 - CLOSE                 | **PASS** | Already marked complete.                                                                                                                                                          |
| S3 - DEPOSIT / WITHDRAW    | **PASS** | Already marked complete.                                                                                                                                                          |
| S4 - Callback Registration | **PASS** | `monitor/manager.go:15-18`:`subscriber` struct stores `addr *net.UDPAddr` + `expiresAt time.Time`. `Register()` stores client on each MONITOR request.                            |
| S4 - Blocking Behavior     | **PASS** | `bankClient.cpp:474-508`:`listen_server()` runs a blocking `while` loop for the full duration; the main `run()` loop cannot accept new input until it returns. No multithreading. |
| S4 - Push Updates          | **PASS** | `handler.go` calls `mon.NotifyAll(...)` after: Open (line 131), Close (line 163), Deposit (line 207), Withdraw (line 249), Transfer. All S1-S3 mutations notify subscribers.      |
| S4 - Expiration / Cleanup  | **PASS** | `manager.go:122-148`:`periodicSweep()` goroutine runs every 5 seconds (configured in `main.go:37`) calling `evictExpired()`. Also lazily cleaned during `NotifyAll()`.            |

**Section result: 7 / 7 PASS**

---

## 3. CUSTOM OPERATIONS & IDEMPOTENCY

| Item                                   | Status      | Evidence                                                                  |
| -------------------------------------- | ----------- | ------------------------------------------------------------------------- |
| Op 5 - Idempotent (Get Balance)        | **PASS**    | Already marked complete.                                                  |
| Op 6 - Non-Idempotent (Transfer)       | **PASS**    | Already marked complete.                                                  |
| Comparison Test (under simulated loss) | **PENDING** | Empirical task:must be run manually and results documented in the report. |

**Section result: 2 / 2 code items PASS:1 item requires manual testing**

---

## 4. FAULT TOLERANCE (INVOCATION SEMANTICS)

| Item                                 | Status   | Evidence                                                                                                                                                                                   |
| ------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| CLI Toggle (At-Least / At-Most)      | **PASS** | Server: `main.go:46`:`semantics.ParseMode(os.Args[1])`, usage `./server <at-least-once\|at-most-once> [loss-rate]`. Client: `semantics.cpp:17-30`::-l`= AT_LEAST_ONCE,`-m` = AT_MOST_ONCE. |
| Loss Simulator                       | **PASS** | `server/internal/semantics/losssim.go`:`ShouldDrop()` uses `rand.Float64() < rate`. Applied at `main.go:127-130` to skip `WriteToUDP` on dropped replies. Loss rate passed as CLI arg.     |
| At-Least-Once (retransmit + timeout) | **PASS** | Already marked complete.                                                                                                                                                                   |
| At-Most-Once - Request IDs           | **PASS** | Already marked complete.                                                                                                                                                                   |
| At-Most-Once - Duplicate Filtering   | **PASS** | Already marked complete.                                                                                                                                                                   |
| At-Most-Once - Reply History         | **PASS** | Already marked complete.                                                                                                                                                                   |

**Section result: 6 / 6 PASS**

---

## 5. INTERFACE & OUTPUT

| Item                                | Status   | Evidence                                                                                                                                                                                           |
| ----------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Client UI (console loop)            | **PASS** | `bankClient.cpp:33-60`:`run()` has an infinite `while(true)` loop: clear screen → print menu → build command → execute → wait for enter.                                                           |
| Client Terminate (Exit option)      | **PASS** | `bankIO.cpp:18`:menu shows `"0. EXIT"`. `bankClient.cpp:64-68`::nput `0` returns `USER_QUIT`, breaking the run loop.                                                                               |
| Client Logging (responses + errors) | **PASS** | Status printed at `bankClient.cpp:434`. Content (account no., balance, callback updates) at lines 421-427, 561-567. Errors logged at all failure points (lines 339, 346, 385, 389, 402, 415, 483). |
| Server Logging - Incoming requests  | **PASS** | `main.go:114`:`log.Printf("[Server] Received %d bytes from %s", n, clientAddr)`. Handler logs each parsed operation in `handler.go` (23 log.Printf calls).                                         |
| Server Logging - Outgoing replies   | **PASS** | `main.go:135`:`log.Printf("[Server] Sent %d byte reply to %s", len(reply), clientAddr)`. Monitor push logged in `manager.go:102-103`.                                                              |

**Section result: 5 / 5 PASS**

---

## 6. REPORT & SUBMISSION

> These are documentation/submission tasks that cannot be verified from the codebase.

| Item                                   | Status                                 |
| -------------------------------------- | -------------------------------------- |
| Work division percentages defined      | Verify in `report.pdf`                 |
| Cover page with all names              | Verify in `report.pdf`                 |
| Packet design (byte-level diagrams)    | Verify in `report.pdf`                 |
| Experiment data (semantics under loss) | Requires running comparison test first |
| 12-page limit                          | Verify in `report.pdf`                 |
| Source code zipped                     | To do before April 2                   |
| Final submission to NTULearn           | To do before April 2 (one rep only)    |

---

## OVERALL SUMMARY

| Section                     | Code Items     | Result                       |
| --------------------------- | -------------- | ---------------------------- |
| Infrastructure & Networking | 7/7            | PASS                         |
| Core Banking Services       | 7/7            | PASS                         |
| Custom Ops & Idempotency    | 2/2 code items | PASS:comparison test pending |
| Fault Tolerance             | 6/6            | PASS                         |
| Interface & Output          | 5/5            | PASS                         |
| Report & Submission         | N/A            | Manual tasks remaining       |

**All implemented code requirements are met.**

### Remaining action items before April 2 deadline:

1. **Run the comparison experiment**:test both semantics under simulated packet loss, record results for the report.
2. **Finalize report**:ensure packet diagrams, work division, and experiment data are included within the 12-page limit.
3. **Zip source code** and submit via NTULearn (one representative per group, no resubmission).
