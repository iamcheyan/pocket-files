#!/usr/bin/env bash
# End-to-end API smoke test against a locally running filesd.
set -euo pipefail
base=${POCKET_FILES_URL:-http://127.0.0.1:8787}
fail=0
check() { local label=$1; shift; if "$@"; then printf 'PASS %s\n' "$label"; else printf 'FAIL %s\n' "$label"; fail=1; fi; }

check health            bash -c "curl -fsS '$base/api/health' | grep -q '\"status\":\"ok\"'"
check storage-list      bash -c "curl -fsS '$base/api/storage' | grep -q '\"tf-a\"'"
check storage-internal  bash -c "curl -fsS '$base/api/storage' | grep -q '\"internal\"'"
check storage-usb       bash -c "curl -fsS '$base/api/storage' | grep -q '\"usb\"'"
check browse-root       bash -c "curl -fsS '$base/api/files?storage_id=internal' | grep -q '\"name\":\"Books\"'"
check browse-folder     bash -c "curl -fsS '$base/api/files?storage_id=internal&path=Documents' | grep -q 'Welcome.txt'"
check search            bash -c "curl -fsS '$base/api/files?storage_id=internal&q=Welcome' | grep -q 'Welcome.txt'"
check ext-filter        bash -c "curl -fsS '$base/api/files?storage_id=internal&path=Documents&ext=txt' | grep -q 'Welcome.txt'"
check traversal-blocked bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' '$base/api/files?storage_id=internal&path=..') = 400"
check no-host-path      bash -c "! curl -fsS '$base/api/storage' | grep -q '/tmp/pocket-files-mock'"
check downloads-state   bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' '$base/api/downloads') = 503"
check fav-list          bash -c "curl -fsS '$base/api/favorites' >/dev/null"
check recent-list       bash -c "curl -fsS '$base/api/recents' >/dev/null"
check settings-get      bash -c "curl -fsS '$base/api/settings' >/dev/null"
check transfers-list    bash -c "curl -fsS '$base/api/transfers' >/dev/null"

if [ "$fail" -ne 0 ]; then echo "API smoke failed"; exit 1; fi
echo "API smoke passed against $base"
