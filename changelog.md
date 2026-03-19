# Changelog

All notable changes to this project will be documented in this file. Have the change log be as detailed and comprehensive as possible for ease of report writing later.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

### Fixed and Improved
2026-03-18 (Jing)

- **UDPSocket Fixes & Enhancement (`client/src/networkUtils/udpSocket.cpp`)**:
  - Implemented `SO_REUSEADDR` to facilitate rapid socket address reuse during testing and deployment, especially on Windows systems.
  - Added optional `should_connect` parameter to the `UDPSocket` constructor to prevent implicit binding on Windows during `connect()`, which previously caused failures for explicitly bound receiver sockets.
  - Implemented `SO_RCVTIMEO` in addition to `SO_SNDTIMEO`, ensuring a 3-second `TIMEOUT` for both sending and receiving operations.
  - Updated `bind_socket()` to refresh `local_ip_port` after a successful bind, ensuring accurate local address reporting via `get_local_info()`.
- **Improved Integration Testing (`client/tests/test_networkUtils.cpp`)**:
  - Updated `NetworkIntegrationTest` to use non-connecting receiver sockets, resolving port binding conflicts on the loopback interface.
  - Verified and finalized 58 high-fidelity unit and integration tests across the client codebase, achieving a 100% pass rate.
- **Protocol Status Handling (`client/include/utils/protocolStatus.h`)**:
  - Added a comprehensive list of `ProtocolStatus` codes (SUCCESS, ACCOUNT_NOT_FOUND, INVALID_CREDENTIALS, etc.) to match the server's business logic requirements.
  - Implemented a centralized `to_string` helper for user-friendly error display in the client UI.
- **Client Request Orchestrator (`client/src/core/bankClient.cpp`)**:
  - Created the `send_to_server(const Protocol::Command&)` function to formalize the end-to-end communication lifecycle.
  - This function encapsulates: command encoding, message construction (including dynamic request ID and local socket info), binary serialization, network transmission with exponential backoff (up to 3 tries), response reception with timeout protection, and automated decoding of server payloads (e.g., balance and account number updates).
- **Protocol Command Encoding (`client/src/protocol/cmdEncoder.cpp`)**:
  - Implemented serialization and deserialization logic for monitoring fields (`monitor_updates` and `monitor_timeout_seconds`), enabling the client to handle account monitoring requests and asynchronous server updates.

### Added
2026-03-16 (Jing)

- **Comprehensive Unit Testing**: Expanded the test suite to 48 tests across 4 test suites, achieving full coverage for critical components. Verified all field encoders, command integrated cycles, message serialization, and client logic.
  - **Test Files**: `test_cmdEncoder.cpp`, `test_msgSerializer.cpp`, `test_bankClient.cpp`, `test_networkUtils.cpp`.
- **Cleaner Command Logic (`client/include/protocol/protocol.h`)**: Refactored `Command` struct and `CommandEncoder` to use a generic `iterate` template with `std::apply`. This eliminates repetitive field handling and ensures consistent processing across encoding and decoding logic.
- **Protocol Layer Abstraction**: Refactored `CommandEncoder` and `MessageSerializer` into a polymorphic design using abstract base classes (`BaseCommandEncoder`, `BaseMessageSerializer`). This improves extensibility and enables easier mocking for unit tests.
- **Dependency Injection Framework (`client/src/core/bankClient.cpp`)**: Refactored `BankClient` to use constructor-based dependency injection. It now depends on interfaces (`BaseSocket`, `BaseCommandEncoder`, `BaseMessageSerializer`) rather than concrete implementations, significantly improving testability and separation of concerns.
- **Robust Client Bootstrap (`client/src/main.cpp`)**: Updated the main entry point to handle server IP and Port via command-line arguments. Implemented a reliable dependency injection sequence in `main` to initialize the client.

### Added
2026-03-02 (Jing)

- **Layered Client Architecture (`client/`)**: Implemented a three-layer client architecture across the application, protocol, and networking layers.
  - **Application Layer (`core/bankClient`)**: Encapsulates the main client loop, user input collection, and service dispatch. Supports all seven banking services (Open, Close, Deposit, Withdraw, Transfer, Monitor, Get Balance) through a menu-driven interface.
  - **Protocol Layer (`protocol/`)**: Handles encoding and decoding of `Command` structs, and serialisation and deserialisation of transport-level `Message` packets, separating business logic from network I/O.
  - **Networking Layer (`networkUtils/`)**: Provides a polymorphic socket abstraction via `BaseSocket` with `UDPSocket` as the concrete implementation, supporting both Windows (Winsock2) and Linux (POSIX).

- **TLV Command Encoding (`client/src/protocol/cmdEncoder.cpp`)**: Implemented a Type-Length-Value (TLV) binary encoding scheme for the `Command` struct.
  - Each field is encoded as `[FieldID (1B)][Length (4B)][Content (NB)]`, supporting variable-length and optional fields without padding.
  - Fields are selectively encoded based on which `std::optional` fields are set, minimising payload size per service type.
  - Network byte order (big-endian) is applied via `htonl`/`ntohl` for all multi-byte integers. A manual byte-swap is used for `double` (monetary value) to ensure cross-platform correctness.

- **Message Serialisation (`client/src/protocol/msgSerializer.cpp`)**: Implemented binary serialisation of the transport-level `Message` struct.
  - Encodes the packet header as `[Type(1B)][RequestID(4B)][IPv4(4B)][Port(2B)][StatusCode(2B)][ContentLen(4B)]` followed by the payload content.
  - Deserialisation includes header size validation and integer-overflow-safe payload length checks using `safe_math::safe_add`.

- **UDP Socket (`client/src/networkUtils/udpSocket.cpp`)**: Implemented UDP send and receive using `sendto`/`recvfrom`.
  - Receive buffer pre-allocated to the maximum UDP datagram size (65,535 bytes) and resized to the actual received length.
  - Platform-specific socket APIs are isolated behind preprocessor guards (`_WIN32`) within the `BaseSocket` RAII wrapper, which closes the socket file descriptor on destruction.

- **Error Handling System (`client/include/utils/`)**: Introduced a type-safe error handling system aligned with the [Google C++ Style Guide](https://google.github.io/styleguide/cppguide.html).
  - `InternalError` enum (`Error::InternalError`): Defines all client-side error conditions across all layers — input validation, socket I/O, TLV decoding, and message deserialisation — each with a distinct code.
  - `Result<T, E>` type (`result.h`): A C++17 `std::variant`-based result type. All fallible functions return `Result<T, Error::InternalError>`, making error propagation explicit at each call site.
  - Overflow-safe arithmetic (`helper.h`): `safe_math::safe_add` and `safe_math::safe_minus` template functions guard all offset and length arithmetic against integer overflow during packet parsing.

- **Interactive Banking CLI (`client/src/core/bankClient.cpp`)**: Implemented a terminal-based menu interface for all seven banking services.
  - Displays a formatted service menu using ANSI colour codes.
  - Collects and validates user input per service: account credentials, currency selection (with re-prompt on invalid input), monetary amounts, and transfer destination details.
  - Currency input is case-insensitive and validated against a static `stringToCurrency` map before the request is dispatched.

### Implemented and Improved
2026-02-28 (Jing)
- **Protocol Encoding Layer (`client/src/protocol/cmdEncoder.cpp`)**: Implemented a message encoding and decoding pipeline.
  - **TLV Parsing Logic**: Implemented a Type-Length-Value (TLV) parsing loop with offset management for multi-field packet reconstruction.
  - **Defensive Design**: Integrated input validation to protect against malformed network payloads and buffer overflows.

- **Networking Layer (`client/src/networkUtils/tcpSocket.cpp`)**: Finalised the socket communication interface.
  - **Optimised I/O**: Implemented `send_message` and `receive_message` with buffer resizing and error handling for reliable data transfer.
  - **API Abstraction**: Standardised network parameter casting to ensure unified behaviour across Windows (Winsock) and Linux (POSIX) systems.

### Added 
2026-02-22 (Tim) 
- **Domain Models (`pkg/models/account.go`)**: Established the core data structures, strictly adhering to the specifications in Section 5.2 of the lab manual.
  - `AccountNumber`: Implemented as a server-generated `uint32` integer (initialized at 10000).
  - `HolderName`: Implemented as a standard Go `string` to satisfy the variable-length string requirement.
  - `Password`: Enforced as an `[8]byte` array. This satisfies the fixed-length string requirement and pre-optimizes the data structure for the mandatory manual byte-level marshalling phase.
  - `CurrencyType`: Implemented as a strongly-typed `uint8` enum (`iota` for SGD, USD, EUR) to fulfill the enumerated type requirement while guaranteeing a 1-byte network payload over UDP.
  - `Balance`: Implemented as a `float64` to fulfill the floating-point requirement and prevent financial precision loss.

- **Persistence Layer (`internal/store/memory.go`)**: Developed the server's in-memory storage engine to manage account states.
  - Fulfills the Section 5.3 requirement to maintain bank account information in memory during server execution via a `map[uint32]*models.Account`.
  - **Concurrency Architecture**: Integrated a `sync.RWMutex` to manage thread-safe memory access. This explicitly satisfies the Section 5.2 requirement to "allow multiple clients to monitor updates to the bank accounts concurrently." It utilizes non-blocking `RLock()` for read-heavy monitoring and strict `Lock()` for state-mutating operations.
  - **Encapsulation**: Abstracted the map operations behind pointer receiver methods (`CreateAccount`, `GetAccount`, `UpdateAccount`, `DeleteAccount`) to ensure atomic state transitions and prevent data corruption from the higher network layers.
  - **Standardized Error Handling**: Added `ErrAccountNotFound` to lay the groundwork for returning proper, consistent error messages to the client console interface, as mandated by the core service requirements.
  
forgot which date oops (Tim) 
- **Core Banking Services (`banking/service.go`)**: Implemented the primary business logic interface mapped to the server requirements.
  - `OpenAccount`: Implemented a service that allows a user to open a new account by specifying his name, a password of his choice, the currency type of the account and the initial account balance. This service creates a new account at the server and returns the account number as the result.
  - `CloseAccount`: Implemented a service that allows a user to close an existing account by specifying his name, his account number and the password]. Added validation logic so that in case of incorrect user input (e.g., wrong password; the account number specified by the user is not under his name or does not exist at the server), a proper error message should be returned. Integrated account-level mutex locking (`acc.Mu.Lock()`) prior to deletion to prevent concurrent operational conflicts.
  - `Deposit` & `Withdraw`: Implemented a service that allows a user to deposit/withdraw money into/from an account by specifying his name, his account number, the password, the currency type and amount he would like to deposit/withdraw. Built the logic so that on completion of the service, the updated balance of the account is returned to the user. Added safety checks so that in case of incorrect user input (e.g., wrong password; there is not enough balance in the account for the user to withdraw), a proper error message should be returned. Integrated account-level mutex locking to guarantee atomic balance mutations.
  - **Sentinel Errors**: Defined reusable error variables (`ErrInvalidCredentials`, `ErrAccountMismatch`, `ErrCurrencyMismatch`, `ErrInsufficientFunds`) to standardize how proper error messages are generated and returned for invalid user inputs.
  - **Authentication Helper (`checkAuth`)**: Centralized the validation logic for credentials to ensure consistent security checks across `CloseAccount`, `Deposit`, and `Withdraw` operations before any state changes occur.

2026-03-07 (Tim)
- **Advanced Banking Operations (`banking/service.go`)**: Expanded the service layer with specialized operations to meet distributed system property requirements.
  - `CheckBalance` (Idempotent): Implemented as the required idempotent operation. This service allows a user to query their current balance without modifying the state of the account. It utilizes a read-only path that validates credentials via `checkAuth` but bypasses state-mutating locks, ensuring that multiple identical requests yield the same result without side effects.
  - `Transfer` (Non-Idempotent): Implemented as the required non-idempotent operation. This service facilitates the movement of funds between two distinct accounts. It is inherently non-idempotent because repeating the same request would result in multiple deductions from the source account.
  - **Deadlock Prevention Logic**: Developed a sophisticated locking strategy within the `Transfer` method. By comparing account numbers and locking them in a consistent global order (lowest ID first), the system prevents "deadly embrace" deadlocks that could occur if two users attempted to transfer funds to each other simultaneously.
  - **Atomic Multi-Account Updates**: Ensured that the transfer of funds is atomic by holding both account locks until both the decrement and increment operations are completed and updated in the `MemoryStore`.
  - **Transfer Validations**: Added logic to prevent transfers to the same account and integrated `ErrTransferSameAccount` and `ErrAccountNotFound` to provide the mandatory clear error feedback to the client interface.

### Added

2026-03-16 (Tim)

* **Monitor Manager (`internal/monitor/manager.go`)**: Implemented a thread-safe UDP callback system to handle the real-time account monitoring requirement.
* `Manager & subscriber`: Established the core tracking structures using a `sync.Mutex` to guarantee thread-safe read/writes to the active subscriber map across concurrent UDP requests.
* `Register(clientAddr, interval)`: Added idempotent client registration. It maps the client's `net.UDPAddr` to an absolute `expiresAt` timestamp, allowing clients to cleanly register or extend their monitoring window.
* `NotifyAll(update)`: Implemented the broadcast engine. Optimized to marshal the payload exactly once per update (saving CPU cycles) and features "lazy eviction" to prune expired subscribers dynamically during the iteration loop.
* `periodicSweep()`: Added a dedicated background goroutine that routinely sweeps and deletes expired clients, preventing memory leaks during periods of low transactional activity.
* `MarshalUpdateFunc`: Introduced a dependency injection pattern for payload marshalling. This cleanly decouples the network broadcasting logic from the specific byte-level wire format used by the handler layer.


2026-03-19 (Tim)
Invocation Semantics Package (internal/semantics/): Implemented a fully decoupled fault tolerance layer to support at-least-once and at-most-once invocation semantics.
* `RequestHeader & ParseHeader` (header.go): Defined the 5-byte wire format contract — [ServiceID: uint8][RequestID: uint32 BE] — for all incoming UDP packets. The parser maintains a clean boundary with the marshalling layer by extracting only the header and passing the remaining payload untouched to the handler.
* `ReplyHistory (history.go)`: Built a thread-safe, two-level reply cache (sync.RWMutex over map[clientAddr]map[requestID][]byte) that stores defensive copies of raw reply bytes. It supports Lookup for duplicate detection and EvictBefore for sliding-window eviction based on monotonic request IDs.
* `Dispatcher (dispatcher.go)`: Developed the core semantics engine positioned between the UDP read loop and the handler. It manages the logic for at-most-once mode, checking the ReplyHistory to prevent non-idempotent side effects (like double-deposits) by returning cached replies for duplicate requests.
* `LossSimulator (losssim.go`): Added a configurable packet loss simulator with a dedicated rand.Rand source to avoid global lock contention. It is designed to provoke retransmission scenarios at both request-receive and reply-send points to support reproducible lab report experiments.
