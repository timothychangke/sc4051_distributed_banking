# Changelog

All notable changes to this project will be documented in this file. Have the change log be as detailed and comprehensive as possible for ease of report writing later.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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