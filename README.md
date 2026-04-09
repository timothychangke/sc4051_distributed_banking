# SC4051 Distributed Banking System

A cross-language distributed banking application demonstrating fault-tolerant UDP communication between a C++ client and Go server. This system showcases custom binary protocol design, invocation semantics (at-least-once and at-most-once), and handling of network unreliability.

**Team Members:**

- Timothy Chang (U2220136J)
- O Jing (U2223350G)
- Neo Zhi Xuan (U2222293E)

---

## 📋 System Overview

### Architecture

- **Client:** C++ application with UDP socket support (Windows & POSIX compatible)
- **Server:** Go-based server with concurrent request handling
- **Protocol:** Custom binary protocol over UDP (no RPC framework)
- **Key Features:** Fault tolerance, packet loss simulation, account monitoring with callbacks

### Supported Operations

1. **Open Account** - Create a new account with name, password, currency, and initial balance
2. **Close Account** - Close an existing account
3. **Deposit** - Add funds to an account
4. **Withdraw** - Remove funds from an account
5. **Transfer** - Move funds between accounts
6. **Get Balance** - Query current account balance
7. **Monitor** - Subscribe to real-time account updates via callbacks

### Invocation Semantics

| Semantic          | Server Argument | Client Flag | Description                                     | Retry Behavior                                                     |
| ----------------- | --------------- | ----------- | ----------------------------------------------- | ------------------------------------------------------------------ |
| **At-Least-Once** | `at-least-once` | `-l`        | Guarantees request reaches server at least once | Retransmits until reply received; may cause duplicate processing   |
| **At-Most-Once**  | `at-most-once`  | `-m`        | Guarantees request processed at most once       | Retransmits with duplicate detection; server maintains reply cache |

---

## 🏗️ Prerequisites

### WSL Ubuntu / Linux

- **C++ Build:** CMake 3.15+, GCC/Clang with C++17 support
- **Go Build:** Go 1.22+ (optional)
- **Network:** Port 2222 must be available

---

## 🔨 Building the System

### 1. Build the Server (Go)

```bash
cd server
go mod tidy
go build -o bin/server cmd/main.go
```

**Output:** `server/bin/server`

### 2. Build the Client (C++)

```bash
cd client
mkdir -p build
cd build
cmake ..
cmake --build .
```

**Output:** `client/build/client`

---

## ▶️ Running the System

### Server Startup

The server must be started with the semantics keyword:

```text
./bin/server <at-least-once|at-most-once> [loss-rate]
```

#### At-Least-Once

```bash
./bin/server at-least-once
```

#### At-Most-Once

```bash
./bin/server at-most-once
```

#### With simulated loss

```bash
./bin/server at-least-once 0.3
./bin/server at-most-once 0.5
```

Expected startup output:

```text
[Server] Invocation semantics: at-most-once
[Server] Simulated reply loss rate: 50%
[Server] Listening on 0.0.0.0:2222
```

### Client Startup

The client requires a semantics flag:

```text
./client <Server_IP> <Server_Port> <-l|-m>
```

Examples:

```bash
./client 127.0.0.1 2222 -m
./client 127.0.0.1 2222 -l
```

### Step-by-step correctness test

These flows demonstrate the most important integration-test behaviors. Keep the server running in a separate terminal the whole time.

---

#### Scenario A: at-most-once with no packet loss

**Goal:** show the clean baseline where every request succeeds immediately.

1. Start the server:

```bash
cd server
go build -o bin/server cmd/main.go
./bin/server at-most-once
```

2. Start the client:

```bash
cd client/build
./client 127.0.0.1 2222 -m
```

3. Choose `1` to open an account.

4. Enter:

```text
User name: X
Password: Y
Currency: SGD
Initial balance: 1000
```

5. Choose `3` to deposit.

6. Enter:

```text
Account number: <the account number returned above>
Currency: SGD
Amount: 250
```

7. Choose `6` to check the balance.

**Expected response:** the account opens once, the deposit succeeds once, and the balance becomes `1250`.

**Why this happens:** at-most-once deduplicates repeated requests, but in this case no retry is needed because there is no packet loss.

---

#### Scenario B: at-most-once with packet loss

**Goal:** show that packet loss causes retries, but the server still prevents duplicate processing.

1. Start the server with loss:

```bash
cd server
./bin/server at-most-once 0.5
```

2. Start the client:

```bash
cd client/build
./client 127.0.0.1 2222 -m
```

3. Choose `1` to open an account.

4. Enter:

```text
User name: X
Password: Y
Currency: SGD
Initial balance: 1000
```

5. If the reply is lost, the client should time out and retry automatically.

**Expected response if the packet/reply is lost:**

- the client shows a retry or timeout message
- the server eventually returns the same success reply
- the account is created only once

**Why this happens:** at-most-once stores the reply history using the request ID. When the client retries, the server recognizes the duplicate and returns the cached reply instead of running the request again.

---

#### Scenario C: at-least-once with no packet loss

**Goal:** show the baseline for at-least-once when the network is healthy.

1. Start the server:

```bash
cd server
./bin/server at-least-once
```

2. Start the client:

```bash
cd client/build
./client 127.0.0.1 2222 -l
```

3. Choose `1` to open an account.

4. Enter:

```text
User name: X
Password: Y
Currency: SGD
Initial balance: 1000
```

5. Choose `3` to deposit.

6. Enter:

```text
Account number: <the account number returned above>
Currency: SGD
Amount: 250
```

7. Choose `6` to check the balance.

**Expected response:** the account opens once, the deposit succeeds once, and the balance becomes `1250`.

**Why this happens:** at-least-once works normally when no packet loss occurs because there is no need for retransmission.

---

#### Scenario D: at-least-once with packet loss

**Goal:** show the important failure mode where retries may re-execute a non-idempotent request.

1. Start the server with loss:

```bash
cd server
./bin/server at-least-once 0.5
```

2. Start the client:

```bash
cd client/build
./client 127.0.0.1 2222 -l
```

3. Choose `1` to open an account.

4. Enter:

```text
User name: X
Password: Y
Currency: SGD
Initial balance: 1000
```

5. If the reply is lost, the client should time out and retry automatically.

**Expected response if the packet/reply is lost:**

- the client shows a retry or timeout message
- the request may be executed again on the server
- if the request is non-idempotent, you may see duplicated side effects

For example:

- opening an account may create more than one account if the operation is repeated
- depositing may apply more than once if the same request is executed again

**Why this happens:** at-least-once guarantees delivery by retrying, but it does not prevent re-execution. That is why duplicate effects are possible when a reply is lost and the client resends the request.

---

#### Scenario E: server down / client cannot reach server

**Goal:** show the failure case when the server is not available.

1. Do not start the server.

2. Start the client:

```bash
cd client/build
./client 127.0.0.1 2222 -m
```

3. Choose `1` and enter the account details.

**Expected response:** the client should time out, print an error, and not display a successful account creation or deposit.

**Why this happens:** the socket cannot receive a valid reply from the server, so the request fails after the timeout window.

## 📊 Testing Fault Tolerance

### Recommended Test Scenarios

| Scenario                      | Server Command                   | Expected Behavior                                      |
| ----------------------------- | -------------------------------- | ------------------------------------------------------ |
| **No Loss**                   | `./bin/server at-most-once`      | All operations succeed immediately                     |
| **30% Loss (At-Most-Once)**   | `./bin/server at-most-once 0.3`  | Occasional retries; correct due to duplicate detection |
| **50% Loss (At-Most-Once)**   | `./bin/server at-most-once 0.5`  | Frequent retries; correctness maintained               |
| **30% Loss (At-Least-Once)**  | `./bin/server at-least-once 0.3` | Occasional retries; may see duplicate processing       |
| **High Loss (At-Least-Once)** | `./bin/server at-least-once 0.8` | Many retries; significant risk of duplicates           |
