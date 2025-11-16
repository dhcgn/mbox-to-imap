#!/bin/bash

go run main.go  mbox-to-imap \
	--dry-run \
	--mbox "test_data/All mail Including Spam and Trash.mbox" \
	--imap-host 127.0.0.1 \
	--imap-user test \
	--imap-pass 'qwert' \
	--target-folder DEBUG \
	--log-level info \
	--log-dir "debug_scripte_output" \
	--state-dir "debug_scripte_output" \
	--exclude-header '\nSubject: .*.Google Account\n'

# bash _debug.dry-run_info.sh 
# time=2025-11-15T23:38:17.154+01:00 level=INFO msg="starting mbox-to-imap" mbox="test_data/All mail Including Spam and Trash.mbox" target=DEBUG dryRun=true
# Counting messages: 1.6/1.8 MB (85%) [085/100] ████████████████████████  85% | 0s
#  SUCCESS  Message counting complete in 3.416385ms
#  INFO  Total messages in mbox: 11                                                                                                                                                                                                     
#  INFO  Already processed: 11                                                                                                                                                                                                          
#  INFO  Remaining to process: 0                                                                                                                                                                                                        
#                                                                                                                                                                                                                                       
# Processing messages [12/12] ██████████████████████████████████████████ 100% | 0s 0s
# 
# 
# # Summary Statistics
# 
#  INFO  Duration: 20.239788ms
#  INFO  Scanned: 6
#  INFO  Enqueued: 0
#  INFO  Uploaded: 0
#  INFO  Dry-run uploaded: 0
#  INFO  Duplicates (skipped): 6
#  INFO  Errors: 0
# time=2025-11-15T23:38:17.178+01:00 l	