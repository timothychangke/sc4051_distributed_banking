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