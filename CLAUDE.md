# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Test Commands
- Build: `make build`
- Test: `make test` (all tests) or `go test -v ./internal/path/to/package` (single package)
- Run: `make run` or `./kitedata --flags`
- Docker: `make docker-build` and `make docker-run`
- Config: `make config` (creates config file from example)

## Code Style Guidelines
- Format with `gofmt` before committing
- Use tabs for indentation
- Group imports: standard lib first, then third-party, then internal
- Types before functions in package files
- PascalCase for exported symbols, camelCase for internal ones
- Error handling: explicit checking with early returns, use `fmt.Errorf("message: %w", err)` for wrapping
- Context aware: respect context cancellation with `select` statements
- Tests: table-driven tests with clear input/expected output
- Documentation: always add comments for exported types and functions

## Architecture
- Clean separation of concerns (auth, historical, instruments, config)
- Respect the package hierarchy (no circular dependencies)
- Configuration via config.yaml, env vars, or command-line flags