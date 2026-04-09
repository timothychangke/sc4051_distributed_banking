# Distributed Banking Server

This is the Go UDP server for the distributed banking system.

## Usage

```text
./bin/server <at-least-once|at-most-once> [loss-rate]
```

- `at-least-once` or `at-most-once` is required.
- `loss-rate` is optional and must be between `0.0` and `1.0`.

Examples:

```sh
go build -o bin/server cmd/main.go
./bin/server at-most-once
./bin/server at-least-once 0.3
```

## Setup

1. Install dependencies

```sh
go mod tidy
```

2. Build binary

```sh
go build -o bin/server cmd/main.go
```

3. Run binary

```sh
./bin/server at-most-once
```

4. Check for build errors before pushing

```sh
go build ./...
```
