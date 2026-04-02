# Integration Test Suite: Distributed Banking System

## Overview

This test suite automates the manual testing checklist 

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

