#!/bin/bash

go run main.go  mbox-stats \
	 "test_data/All mail Including Spam and Trash.mbox" \
	 --exclude-header '\nSubject: .*.Google Account\n'  \
	 --output "debug_scripte_output"
