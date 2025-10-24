# mbox-to-imap

**mbox-to-imap** is a command-line tool written in **Go** for importing emails from an `mbox` file into an IMAP mailbox.  
It supports **repeatable, incremental synchronization** — only new, unsynchronized messages are transferred when run multiple times.  
It also includes a **strict filtering system** (include **or** exclude) and an enhanced **dry-run** mode with rich stats.

Ideal for importing **Google Takeout** archives or migrating email data between providers.

---

## 🚀 Overview

Use `mbox-to-imap` to transfer emails from `.mbox` files (e.g., from Google Takeout) into another IMAP-compatible mailbox.

Core features:
- Reliable `.mbox` → IMAP import
- Incremental sync (idempotent)
- Stop and resume safely
- **Filtering via include *or* exclude (mutually exclusive) with regex rules**
- **Dry-run analysis** with counts and top senders/recipients/subjects/subject-word histogram

---

## ⚙️ Features

- **Incremental synchronization:** Avoid duplicate uploads
- **Graceful resume:** Continue after interruption
- **Allow/deny filters:** Include-only (allow list) *or* exclude-some (block list)
- **Dry-run statistics:** Preview and verify filters; always outputs a stats summary
- **Cross-platform:** Runs on Windows, macOS, Linux
- **Lightweight:** No database, uses a local state file to track transferred messages.

---

## 🧩 Installation

You need **Go 1.20+**.

```bash
git clone https://example.org/your-org/mbox-to-imap.git
cd mbox-to-imap
go build -o mbox-to-imap ./cmd/mbox-to-imap
````

---

## 💡 Basic Usage

### Import MBOX to IMAP

```bash
./mbox-to-imap \
  --mbox /path/to/AllMail.mbox \
  --imap-host imap.example.com \
  --imap-port 993 \
  --imap-user user@example.com \
  --imap-pass "secret" \
  --target-folder "INBOX/Imported"
```

### Dry Run (Simulation + Stats)

Simulate synchronization, apply (optional) filters, and show **summary statistics**:

```bash
./mbox-to-imap \
  --mbox archive.mbox \
  --imap-host imap.example.com \
  --imap-user user@example.com \
  --imap-pass secret \
  --dry-run
```

Example (illustrative) dry-run output:

```
[Dry Run Summary]
Total messages scanned: 3500
Matched (after filters): 2300
Skipped (excluded or not allowed): 1200
Would upload: 2300

Top 10 Senders (From):
1) alice@example.com (410)
2) newsletters@service.com (350)
...

Top 10 Recipients (To):
1) me@example.com (1,540)
2) team@example.com (280)
...

Top 10 Subjects:
1) "Project X weekly sync" (42)
2) "Invoice April" (31)
...

Top 100 words in subjects (space-separated tokens):
project(120), invoice(85), april(50), weekly(48), sync(45), ...
```

> Tokenization for the “Top 100 words in subjects” is space-based; punctuation handling may evolve.

---

## 🔧 Command-Line Options

| Flag                     | Description                                          | Default / Required      |
| ------------------------ | ---------------------------------------------------- | ----------------------- |
| `--mbox`                 | Path to `.mbox` file                                 | **required**            |
| `--imap-host`            | IMAP server hostname                                 | **required**            |
| `--imap-port`            | IMAP port                                            | `993`                   |
| `--imap-user`            | IMAP username                                        | **required**            |
| `--imap-pass`            | IMAP password (or use env var)                       | **required**            |
| `--use-tls`              | Use TLS for IMAP connection                          | `true`                  |
| `--insecure-skip-verify` | Skip TLS certificate validation                      | *not recommended*       |
| `--target-folder`        | Target IMAP folder for imported mail                 | `INBOX`                 |
| `--state-dir`            | Directory for state files                            | `~/.mbox-to-imap/state` |
| `--concurrency`          | Number of parallel uploads                           | `1`                     |
| `--dry-run`              | Simulate sync and print stats (no changes)           | `false`                 |
| `--log-level`            | Logging verbosity (`debug`, `info`, `warn`, `error`) | `info`                  |

### Filtering (mutually exclusive modes)

> You can use **include** *or* **exclude**, **not both**.
> All filter values are **regex**.
> “Header” checks search across the raw header block (e.g., `From:`, `To:`, `Cc:`, `Subject:` etc.).
> “Body” checks search the message body text.

**Include-only (allow list):** Only messages matching at least one include rule are considered; all others are **dropped**.

* `--include-header "<regex>"`
* `--include-body "<regex>"`

**Exclude-some (block list):** All messages are included **except** those matching any exclude rule.

* `--exclude-header "<regex>"`
* `--exclude-body "<regex>"`

You may provide each flag multiple times to add multiple rules (implementation-dependent; or use `|` in a single regex).

---

## 🧱 Filtering Examples

### Allow list (include-only)

Only include mails from a domain **and** with “Project X” in the subject or body:

```bash
./mbox-to-imap \
  --mbox mails.mbox \
  --imap-host imap.server.com \
  --imap-user user@domain.com \
  --imap-pass "$IMAP_PASS" \
  --include-header "From:\s*.*@trusted\.com" \
  --include-header "Subject:\s*.*Project X.*"
# Using include mode means only messages matching ANY include regex are kept.
```

### Block list (exclude-some)

Exclude newsletters and mails with “unsubscribe” in the body:

```bash
./mbox-to-imap \
  --mbox mails.mbox \
  --imap-host imap.server.com \
  --imap-user user@domain.com \
  --imap-pass "$IMAP_PASS" \
  --exclude-header "Subject:\s*.*(newsletter|promo).*" \
  --exclude-body "(?i)unsubscribe"
```

> **Rule semantics:**
>
> * **Include mode:** keep if `include-header` **OR** `include-body` matches (allow list). Everything else is dropped.
> * **Exclude mode:** drop if `exclude-header` **OR** `exclude-body` matches; otherwise keep.

---

## 🔁 Incremental Synchronization

* Each `.mbox` sync uses a **state file** to track transferred messages.
* Already-synced messages (by `Message-ID`) are skipped.
* If the tool is stopped mid-transfer, it resumes from the last checkpoint.

Mechanism:

1. Identify messages by `Message-ID`.
2. Record successfully transferred message IDs in a state file.
3. On subsequent runs, skip messages whose IDs are in the state file.

---

## 🧪 Logging & Stats

* `--dry-run` **always** produces a stats summary:

  * Total scanned / matched / skipped / would-upload
  * **Top 10 senders (From)**
  * **Top 10 recipients (To)**
  * **Top 10 subjects**
  * **Top 100 subject words** (space-separated tokenization)
* `--log-level debug` enables message-by-message tracing

---

## 🔒 Security Recommendations

* Always back up `.mbox` files before running imports.
* Use environment variables or app passwords:

  ```bash
  export IMAP_PASS="secret"
  ./mbox-to-imap --imap-pass "$IMAP_PASS"
  ```
* Keep state files private (may contain metadata).
* Avoid `--insecure-skip-verify` except for debugging in trusted environments.

---

## ⚠️ Disclaimer

This software is provided **“as is”** without any warranty.
Use at your own risk — the authors assume **no liability** for any damage or data loss.
Always test with a small sample before large migrations.

---

## 🧑‍💻 Development

Contributions are welcome!

```bash
# Run tests
go test ./...
```

Please format code (`go fmt`) and lint before submitting PRs.

---

## 🪪 License

MIT License — see `LICENSE`.
