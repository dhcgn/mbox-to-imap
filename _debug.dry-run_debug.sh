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
	--state-dir "debug_scripte_output" \
	--exclude-header '\nSubject: .*.Google Account\n'

# bash _debug.dry-run_debug.sh 
# time=2025-11-15T23:38:56.983+01:00 level=INFO msg="starting mbox-to-imap" mbox="test_data/All mail Including Spam and Trash.mbox" target=DEBUG dryRun=true
# time=2025-11-15T23:38:56.983+01:00 level=DEBUG msg="counting messages in mbox file" path="test_data/All mail Including Spam and Trash.mbox"
# time=2025-11-15T23:38:56.986+01:00 level=DEBUG msg="counted messages" total=11
# time=2025-11-15T23:38:56.986+01:00 level=DEBUG msg="state tracker loaded" alreadyProcessed=11
# time=2025-11-15T23:38:57.009+01:00 level=INFO msg="stats summary" scanned=11 enqueued=0 uploaded=0 dryRunUploaded=0 duplicates=11 errors=0 duration=22.969166ms
# time=2025-11-15T23:38:57.010+01:00 level=INFO msg="pipeline completed" duration=23.191609ms
