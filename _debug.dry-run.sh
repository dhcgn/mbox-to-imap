#!/bin/bash

go run main.go \
	--dry-run \
	--mbox "/home/d/Downloads/takeout-20251017T122523Z-1-001/Takeout/Mail/All mail Including Spam and Trash.mbox" \
	--imap-host 127.0.0.1 \
	--imap-user test \
	--imap-pass 'qwert' \
	--target-folder DEBUG \
	--log-level debug \
	--log-dir "." \
	--state-dir "."