## mbox-to-imap Copilot Guide

### Repository Snapshot
- Purpose: Go CLI that ingests `.mbox` archives, streams messages through a producer/consumer pipeline, and uploads to IMAP with dry-run stats, regex filters, and incremental resume via state files.
- Footprint: Single Go module (~15 packages) plus sample `.mbox` fixtures and debug artifacts; everything lives under the repo root (no vendoring).
- Primary tech: Go 1.25, Cobra CLI, go-imap v2 client, go-mbox parser, pterm for progress UI.

### Toolchain & Environment
- Verified with `go version` → `go1.25.4 linux/amd64`. Keep Go ≥1.25 to match CI (`actions/setup-go@v4`).
- Module path `github.com/dhcgn/mbox-to-imap`; dependencies declared in `go.mod` / `go.sum`. No extra services required unless you run a real IMAP target.
- Set `IMAP_PASS` when avoiding the `--imap-pass` flag. TLS is on by default; `--insecure-skip-verify` exists only for trusted test servers.

### Required Workflow (Bootstrap → Ship)
1. **Bootstrap**: run `go version` to confirm toolchain; ensure sample data exists (`test_data/All mail Including Spam and Trash.mbox`, `test_data/corrupted.mbox`).
2. **Format**: `go fmt ./...` (fast, no extra setup).
3. **Build**: `go build ./...` (passes cleanly; no artifacts generated beyond the module cache).
4. **Test**: `go test ./...` (current runtime ≈ 0.01s; exercises `filter`, `mbox` using embedded fixtures). No network requirements.
5. **Smoke CLI**: `go run . --help`, `go run . mbox-to-imap --help`, `go run . mbox-stats --help` to confirm flag registration if you touch Cobra setup.
6. **Dry-run demo** *(optional but mirrors CI)*: `bash _debug.dry-run_debug.sh` (uses sample mbox, skips uploads, writes logs into `debug_scripte_output/`).
7. **Stats demo** *(optional but mirrors CI)*: `bash _debug.mbox-stats.sh` and `_debug.mbox-stats_filtered.sh` to regenerate CSV summaries. These rely on the large sample `.mbox` file and will overwrite `debug_scripte_output*/report_*.csv`.
8. **Real import**: only attempt with a reachable IMAP host. Run something like `go run . mbox-to-imap --mbox <file> --imap-host <host> --imap-user <user> --imap-pass <pass> --target-folder <folder> --state-dir <dir>`. Expect network/tls failures without a live server.

Always execute steps 2–4 before sending code for review. Steps 6–7 are required whenever you change filtering, stats, or progress handling because CI invokes those scripts.

### Debug & State Artifacts
- State tracking uses `state.FileTracker` writing `processed.jsonl` under `~/.mbox-to-imap/state` (or `--state-dir`). Delete this file to force reprocessing but expect duplicates.
- Debug scripts dump CSVs into `debug_scripte_output/` and `debug_scripte_output_filtered/` plus logs named `mbox-to-imap-<timestamp>.log`. These folders are git-ignored; feel free to clean between runs.

### CI Expectations
- `.github/workflows/go.yml` (push & PR to `master`) runs, in order: checkout → Go 1.25 → `go build ./...` → `go test ./...` → the four `_debug.*.sh` scripts. Locally reproduce by running the same commands sequentially.
- `.github/workflows/release.yml` builds Windows/Linux binaries on tag pushes (`v*`) with ldflags filling `main.Version`, `CommitID`, `BuildTime`, then uploads them via `softprops/action-gh-release`.

### Project Layout Highlights
- `main.go`: CLI entry, prints version info from ldflags, delegates to `cmd` package.
- `cmd/root.go`: Cobra root plus `mbox-to-imap` subcommand, wiring for logger setup, pipeline assembly.
- `cmd/mbox-stats.go`: Standalone stats command producing CSVs and live console summaries.
- `config/config.go`: Flag registration, env fallbacks, validation, default state directory helper.
- `mbox/`: File reader, message parsing, producer wiring, count helper, tests with embedded `corrupted.mbox`.
- `filter/`: Regex filtering logic with hit tracking, unit tests cover include/exclude and header/body modes.
- `imap/`: IMAP uploader using go-imap v2, handles dry-run short-circuit, folder creation, append pipeline.
- `runner/`: Pipeline coordinator (stages, channels, event stream, state tracker binding).
- `stats/`, `progress/`: Event aggregation, console reporting, pterm progress bars.
- `state/`: Memory + file-backed tracker for processed message hashes.
- `test_data/`: Sample `.mbox` fixtures required by scripts/tests; keep paths stable.
- `_debug*.sh`: Reusable scenarios relied on by CI. Modify scripts and README together if behavior changes.
- `scripts/increase_sem_version_git_tag.ps1`: Release helper for manual tagging (no automation depends on it).

### Operational Notes
- Most packages depend on `slog` for structured logging; adjust log levels via `--log-level` (debug/info/warn/error).
- Producer/consumer pipeline relies on buffered channels (size 32) and stage registration order; watch for deadlocks if you change queue depths.
- The pipeline skips messages where a `Message-ID` cannot be extracted. Tests cover corrupted input behavior—maintain that contract.
- Progress bars (info log level) require a TTY; expect plain logs when running in non-interactive environments (e.g., GitHub Actions).

### Toolchain & Environment
- Verified with `go version` → `go1.25.4 linux/amd64`. Keep Go ≥1.25 to match CI (`actions/setup-go@v4`).
- Module path `github.com/dhcgn/mbox-to-imap`; dependencies declared in `go.mod` / `go.sum`. No extra services required unless you run a real IMAP target.
- Set `IMAP_PASS` when avoiding the `--imap-pass` flag. TLS is on by default; `--insecure-skip-verify` exists only for trusted test servers.

### Required Workflow (Bootstrap → Ship)
1. **Bootstrap**: run `go version` to confirm toolchain; ensure sample data exists (`test_data/All mail Including Spam and Trash.mbox`, `test_data/corrupted.mbox`).
2. **Format**: `go fmt ./...` (fast, no extra setup).
3. **Build**: `go build ./...` (passes cleanly; no artifacts generated beyond the module cache).
4. **Test**: `go test ./...` (current runtime ≈ 0.01s; exercises `filter`, `mbox` using embedded fixtures). No network requirements.
5. **Smoke CLI**: `go run . --help`, `go run . mbox-to-imap --help`, `go run . mbox-stats --help` to confirm flag registration if you touch Cobra setup.
6. **Dry-run demo** *(optional but mirrors CI)*: `bash _debug.dry-run_debug.sh` (uses sample mbox, skips uploads, writes logs into `debug_scripte_output/`).
7. **Stats demo** *(optional but mirrors CI)*: `bash _debug.mbox-stats.sh` and `_debug.mbox-stats_filtered.sh` to regenerate CSV summaries. These rely on the large sample `.mbox` file and will overwrite `debug_scripte_output*/report_*.csv`.
8. **Real import**: only attempt with a reachable IMAP host. Run something like `go run . mbox-to-imap --mbox <file> --imap-host <host> --imap-user <user> --imap-pass <pass> --target-folder <folder> --state-dir <dir>`. Expect network/tls failures without a live server.

Always execute steps 2–4 before sending code for review. Steps 6–7 are required whenever you change filtering, stats, or progress handling because CI invokes those scripts.

### Debug & State Artifacts
- State tracking uses `state.FileTracker` writing `processed.jsonl` under `~/.mbox-to-imap/state` (or `--state-dir`). Delete this file to force reprocessing but expect duplicates.
- Debug scripts dump CSVs into `debug_scripte_output/` and `debug_scripte_output_filtered/` plus logs named `mbox-to-imap-<timestamp>.log`. These folders are git-ignored; feel free to clean between runs.

### CI Expectations
- `.github/workflows/go.yml` (push & PR to `master`) runs, in order: checkout → Go 1.25 → `go build ./...` → `go test ./...` → the four `_debug.*.sh` scripts. Locally reproduce by running the same commands sequentially.
- `.github/workflows/release.yml` builds Windows/Linux binaries on tag pushes (`v*`) with ldflags filling `main.Version`, `CommitID`, `BuildTime`, then uploads them via `softprops/action-gh-release`.

### Project Layout Highlights
- `main.go`: CLI entry, prints version info from ldflags, delegates to `cmd` package.
- `cmd/root.go`: Cobra root plus `mbox-to-imap` subcommand, wiring for logger setup, pipeline assembly.
- `cmd/mbox-stats.go`: Standalone stats command producing CSVs and live console summaries.
- `config/config.go`: Flag registration, env fallbacks, validation, default state directory helper.
- `mbox/`: File reader, message parsing, producer wiring, count helper, tests with embedded `corrupted.mbox`.
- `filter/`: Regex filtering logic with hit tracking, unit tests cover include/exclude and header/body modes.
- `imap/`: IMAP uploader using go-imap v2, handles dry-run short-circuit, folder creation, append pipeline.
- `runner/`: Pipeline coordinator (stages, channels, event stream, state tracker binding).
- `stats/`, `progress/`: Event aggregation, console reporting, pterm progress bars.
- `state/`: Memory + file-backed tracker for processed message hashes.
- `test_data/`: Sample `.mbox` fixtures required by scripts/tests; keep paths stable.
- `_debug*.sh`: Reusable scenarios relied on by CI. Modify scripts and README together if behavior changes.
- `scripts/increase_sem_version_git_tag.ps1`: Release helper for manual tagging (no automation depends on it).

### Operational Notes
- Most packages depend on `slog` for structured logging; adjust log levels via `--log-level` (debug/info/warn/error).
- Producer/consumer pipeline relies on buffered channels (size 32) and stage registration order; watch for deadlocks if you change queue depths.
- The pipeline skips messages where a `Message-ID` cannot be extracted. Tests cover corrupted input behavior—maintain that contract.
- Progress bars (info log level) require a TTY; expect plain logs when running in non-interactive environments (e.g., GitHub Actions).

### Working Style Expectations
- Trust this guide. Search the tree or run exploratory commands only if something here appears incorrect or the repo layout changes.
- When you touch CLI flags or pipeline stages, rerun the relevant debug scripts so generated CSVs/logs stay consistent with CI.
- Prefer small, well-commented changes; add targeted unit tests whenever you alter filtering, state tracking, or parser behavior.
