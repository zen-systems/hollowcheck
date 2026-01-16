# hollowcheck

Quality gate for AI-generated code. Detects stubs, mock data, and hollow implementations.

## The Problem

AI coding tools produce code that compiles but doesn't work. The model satisfies the prompt's surface requirements while deferring actual implementation:

- `TODO: implement this` scattered throughout
- `panic("not implemented")` in every error path
- Hardcoded `example.com` URLs and `user-1, user-2, user-3` test data
- Functions that accept arguments and ignore them
- Linear code where branching logic was clearly needed

The code looks complete. It passes a syntax check. It might even pass a cursory review. But it's hollow—the shape of a solution without the substance.

## The Broader Context

AI-generated content is burying signal under volume. Labs will eventually fix model collapse because it degrades their products. But the "discovery problem"—distinguishing real implementations from hollow ones—has no economic owner.

This tool is infrastructure for that gap. It makes "done" mean something verifiable.

## Quick Start

```bash
go install github.com/zen-systems/hollowcheck@latest

hollowcheck init                              # creates hollowcheck.yaml
hollowcheck lint ./src --contract hollowcheck.yaml
```

## What It Detects

| Category | Examples | Points |
|----------|----------|--------|
| **Missing files** | Required files not present | 20 |
| **Missing symbols** | Expected functions/types not defined | 15 |
| **Forbidden patterns** | `TODO`, `FIXME`, `panic("not implemented")` | 10 |
| **Low complexity** | Linear code where branching was expected | 10 |
| **Missing tests** | Required test functions not present | 5 |
| **Mock data** | `example.com`, sequential IDs, `lorem ipsum` | 3 |

Mock data detection skips `*_test.go` files by default—test fixtures are expected there.

## Hollowness Score

Score is 0-100, calculated from weighted violations. Lower is better.

| Grade | Score | Meaning |
|-------|-------|---------|
| A | 0-10 | Clean |
| B | 11-25 | Minor issues |
| C | 26-50 | Needs work |
| D | 51-75 | Significant gaps |
| F | 76-100 | Hollow |

Default threshold is 25 (B grade). Fail the build if score exceeds threshold.

## Contract Templates

```bash
hollowcheck init --list
```

| Template | Use Case |
|----------|----------|
| `minimal` | Bare minimum quality gate (default) |
| `crud-endpoint` | REST API with database operations |
| `cli-tool` | Command-line tool with subcommands |
| `client-sdk` | API client library with auth/retry |
| `worker` | Background job processor |

```bash
hollowcheck init --template client-sdk --output contracts/sdk.yaml
```

## Contract Format

```yaml
version: "1.0"
name: "my-service"

required_files:
  - path: "cmd/server/main.go"
    required: true

required_symbols:
  - name: "Run"
    kind: function
    file: "cmd/server/main.go"

forbidden_patterns:
  - pattern: "TODO"
    description: "Work-in-progress marker"
  - pattern: 'panic\("not implemented"\)'
    description: "Stub implementation"

mock_signatures:
  skip_test_files: true
  patterns:
    - pattern: 'example\.com'
      description: "Placeholder domain"
    - pattern: "12345|00000"
      description: "Sequential fake IDs"

complexity:
  - symbol: "ProcessRequest"
    file: "internal/handler.go"
    min_complexity: 4

required_tests:
  - name: "TestProcessRequest"
    file: "internal/handler_test.go"
```

## CI Integration

### GitHub Actions

```yaml
name: Quality Gate

on: [pull_request]

jobs:
  hollowcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install hollowcheck
        run: go install github.com/zen-systems/hollowcheck@latest

      - name: Run hollowcheck
        run: hollowcheck lint ./... --contract hollowcheck.yaml
```

The command exits non-zero if hollowness exceeds threshold.

## License

MIT
