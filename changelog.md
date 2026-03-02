# Changelog

All notable changes to this project will be documented in this file. Have the change log be as detailed and comprehensive as possible for ease of report writing later.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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