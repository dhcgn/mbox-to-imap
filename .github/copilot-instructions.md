# mbox-to-imap Copilot Onboarding

## Repository Summary
- Purpose: CLI utility intended to import messages from `.mbox` archives into a target IMAP mailbox with optional filtering, dry-run stats, and incremental sync (per `README.md`).
- Current state: minimal Go stub (`main.go` only contains an empty `main()`); no `go.mod` or functional implementation yet.
- Languages & tooling: Go (verified with `go version go1.24.5 linux/amd64`); no third-party dependencies committed.
- Repository size: tiny; repo root currently contains `README.md`, `main.go`, and this `.github` directory.

## Environment & Tooling
- Install Go ≥1.20 and ensure it is on PATH. No other runtimes or package managers are required right now.
- Module mode is not initialized. Plan to run `go mod init <module-path>` (and later `go mod tidy`) before attempting any real builds or tests.

## Build, Test, Run, Lint
- **Bootstrap**: verify Go is available with `go version`. No additional setup scripts exist.
- **Build**: `go build ./...` fails with `pattern ./...: directory prefix . does not contain main module or its selected dependencies` because `go.mod` is missing. Always create `go.mod` before building.
- **Test**: `go test ./...` fails for the same reason as the build. Once the module exists, this becomes the default test command (no tests exist yet).
- **Run**: `go run .` cannot succeed until a module and actual CLI code exist. After implementation, this will be the quickest way to exercise the tool.
- **Lint/Format**: run `go fmt ./...` prior to submission; no other linters are configured.
- **General guidance**: after adding dependencies, always follow `go mod init` with `go mod tidy` so new modules stay reproducible.

## Project Layout & Key Files
- `README.md`: describes the planned CLI, flags, filtering behavior, dry-run stats, and incremental sync strategy.
- `main.go`: currently just an empty `main()`; serves as the entry point for future CLI wiring.
- `.github/copilot-instructions.md`: this onboarding guide.
- Architectural expectation: implement a producer/consumer pipeline where the MBOX reader (producer) emits `Mail` objects onto a channel and the IMAP uploader (consumer) drains that channel to send messages. Keep the stages decoupled so the mailbox parsing and IMAP transport can be developed and tested independently.

## Validation & CI Expectations
- No GitHub Actions or CI workflows are defined. Reviewers will run `go test ./...` and `go build ./...` locally once a module exists.
- If you add CI, document the workflow commands here so others can reproduce them locally.

## Working Tips for Future Changes
- Expect to add standard Go scaffolding (`go.mod`, `cmd/mbox-to-imap`, `internal/...`) as you begin implementation.
- Preserve clear interfaces between the producer and consumer sides so unit tests can mock IMAP or MBOX pieces independently.
- Honour README promises: incremental sync via state files keyed by `Message-ID`, mutually exclusive include/exclude regex filters, TLS IMAP connectivity, and rich dry-run statistics.
- Update the README and this file whenever build steps, dependencies, or architectural expectations change.

## Exploration Guidance
- Trust these instructions first. Only perform additional searches or exploratory commands if the repository state diverges from what is recorded here.
