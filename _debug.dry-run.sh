#!/bin/bash

go run main.go  mbox-to-imap \
	--dry-run \
	--mbox "test_data/All mail Including Spam and Trash.mbox" \
	--imap-host 127.0.0.1 \
	--imap-user test \
	--imap-pass 'qwert' \
	--target-folder DEBUG \
	--log-level debug \
	--log-dir "debug_scripte_output" \
	--state-dir "debug_scripte_output"