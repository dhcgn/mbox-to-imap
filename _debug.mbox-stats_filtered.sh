#!/bin/bash

go run main.go  mbox-stats \
	 "test_data/All mail Including Spam and Trash.mbox" \
	 --exclude-header '\nSubject: .*.Google Account\n'  \
	 --output "debug_scripte_output"

# bash _debug.mbox-stats_filtered.sh 
# Analyzing mbox file: test_data/All mail Including Spam and Trash.mbox
# 
# Processed 10 messages (skipped 1 by filters, 9.09%)...
# 
# Exclude Header Filters:
#   ✓ \nSubject: .*.Google Account\n: 1 hits
# 
# ---
# 
# Top 10 Delivered-To:
# 1. debugfortakeout@gmail.com (7)
# 
# Top 10 Subject:
# 1. Delivery Status Notification (Failure) (3)
# 2. Test 002 (1)
# 3. FB-56738 is your confirmation code (1)
# 4. Test 001 (1)
# 5. Welcome to Facebook (1)
# 6. 487838 ist dein =?UTF-8?Q?Best=C3=A4tigungscode?= (1)
# 7. Test 003 (1)
# 8. =?UTF-8?B?8J+RqOKAjfCfmoAgV0lDSFRJRzogUmVhZHkgZsO8cg==?= AI? Zuerst =?UTF-8?Q?best=C3=A4tigen=2E=2E=2E?= (1)
# 
# Top 10 From:
# 1. Mail Delivery Subsystem <mailer-daemon@googlemail.com> (3)
# 2. debugfortakeout <debugfortakeout@gmail.com> (3)
# 3. "Facebook" <registration@facebookmail.com> (2)
# 4. TikTok <noreply@account.tiktok.com> (1)
# 5. AInauten <hi@ainauten.com> (1)
# 
# Top 10 To:
# 1. debugfortakeout@gmail.com (4)
# 2. Debug Debug <debugfortakeout@gmail.com> (2)
# 3. test_001@example.org (1)
# 4. test_003@example.org (1)
# 5. "debugfortakeout@gmail.com" <debugfortakeout@gmail.com> (1)
# 6. test_002@example.org (1)
# 
# 
# Reports saved to directory: debug_scripte_output
# 