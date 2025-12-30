# Fuzz Testing for Fetcher Project

This directory contains fuzz tests for the Fetcher project using Go 1.18+ native fuzzing.

## Directory Structure

```
tests/fuzz/
├── manager/
│   ├── connection/     # Connection API fuzz tests
│   ├── fetcher/        # Fetcher API fuzz tests
│   └── schema/         # Schema validation fuzz tests
├── worker/
│   └── message/        # RabbitMQ message parsing fuzz tests
└── shared/
    ├── generators/     # Shared fuzzing utilities
    └── mocks/          # Mock implementations
```

## Running Fuzz Tests

### Run all fuzz tests (30 seconds each):
```bash
make fuzz-all
```

### Run specific component:
```bash
make fuzz-connection   # Connection API tests
make fuzz-fetcher      # Fetcher API tests
make fuzz-schema       # Schema validation tests
make fuzz-message      # Worker message tests
```

### Run with custom duration:
```bash
make fuzz-all FUZZ_TIME=60s
```

### Run individual fuzz test:
```bash
go test -fuzz=FuzzConnectionInputParsing -fuzztime=30s ./tests/fuzz/manager/connection/
```

## Coverage Areas

### Manager Component
- JSON parsing edge cases
- UUID validation (path params, headers)
- Enum validation (DBType)
- Boundary testing (port, string lengths, limits)
- Nested structure validation (SSL, Metadata)
- Unknown fields detection
- Query parameter validation

### Worker Component
- Message payload parsing
- Regex fallback extraction
- FilterCondition operators
- Message header parsing

## Seed Corpus

Each fuzz test includes seed inputs covering:
- Valid minimal inputs
- Boundary values (min, max, min-1, max+1)
- Empty/null values
- Malformed JSON
- Type mismatches
- Unknown fields

## Adding New Fuzz Tests

1. Create test file with `_fuzz_test.go` suffix
2. Use `//go:build go1.18` build tag
3. Add seed corpus with `f.Add()`
4. Test should not panic on any input

## Failure Recovery

### If fuzz test finds a crash:
1. The failing input is saved to `testdata/fuzz/<FuzzName>/`
2. Reproduce with: `go test -run=FuzzName/crashfile ./path/to/test/`
3. Fix the code, run test again to verify
4. Keep the crash file as regression test

### If test hangs:
1. Reduce `FUZZ_TIME` to identify problematic test
2. Check for infinite loops in fuzzed code
3. Add timeout to individual operations if needed
