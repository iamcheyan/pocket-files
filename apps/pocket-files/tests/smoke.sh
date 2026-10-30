#!/usr/bin/env bash
# End-to-end API smoke test against a locally running filesd.
# Every check exercises the real HTTP surface; nothing is stubbed.
set -euo pipefail
base=${POCKET_FILES_URL:-http://127.0.0.1:8787}
fail=0
check() { local label=$1; shift; if "$@"; then printf 'PASS %s\n' "$label"; else printf 'FAIL %s\n' "$label"; fail=1; fi; }
code() { curl -s -o /dev/null -w '%{http_code}' "$@"; }
body() { curl -fsS "$@"; }

check health            bash -c "body() { curl -fsS \"\$@\"; }; body '$base/api/health' | grep -q '\"status\":\"ok\"'"
check storage-list      bash -c "curl -fsS '$base/api/storage' | grep -q '\"tf-a\"'"
check storage-internal  bash -c "curl -fsS '$base/api/storage' | grep -q '\"internal\"'"
check storage-usb       bash -c "curl -fsS '$base/api/storage' | grep -q '\"usb\"'"
check storage-detail    bash -c "curl -fsS '$base/api/storage/internal' | grep -q '\"friendly_name\"'"
check storage-usage     bash -c "curl -fsS '$base/api/storage/internal/usage' | grep -q '\"free\"'"
check detected-readonly bash -c "curl -fsS '$base/api/storage/detected' | grep -q '\"mode\":\"read_only\"'"
check browse-root       bash -c "curl -fsS '$base/api/files?storage_id=internal' | grep -q '\"name\":\"Books\"'"
check browse-folder     bash -c "curl -fsS '$base/api/files?storage_id=internal&path=Documents' | grep -q 'Welcome.txt'"
check search            bash -c "curl -fsS '$base/api/files?storage_id=internal&q=Welcome' | grep -q 'Welcome.txt'"
check ext-filter        bash -c "curl -fsS '$base/api/files?storage_id=internal&path=Documents&ext=txt' | grep -q 'Welcome.txt'"
check sort-order        bash -c "curl -fsS '$base/api/files?storage_id=internal&sort=name&order=desc' | head -c 200 | grep -q 'Videos'"
check pagination        bash -c "curl -fsS '$base/api/files?storage_id=internal&limit=2&page=0' | grep -Eq '\"total\":[6-9]'"
check page-two          bash -c "curl -fsS '$base/api/files?storage_id=internal&limit=2&page=1' | grep -q '\"page\":1'"
check images-folder     bash -c "curl -fsS '$base/api/files?storage_id=internal&path=Images' | grep -q 'Sample Pixel.png'"
check metadata          bash -c "curl -fsS '$base/api/files/internal%3ADocuments%2FWelcome.txt/metadata' | grep -q '\"name\":\"Welcome.txt\"'"
check preview-type      bash -c "test \"\$(curl -s -o /dev/null -w '%{content_type}' '$base/api/files/internal%3ADocuments%2FWelcome.txt/preview')\" = 'text/plain; charset=utf-8'"
check traversal-blocked bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' '$base/api/files?storage_id=internal&path=..') = 400"
check unknown-storage   bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' '$base/api/files?storage_id=nope') = 400"
check no-host-path      bash -c "! curl -fsS '$base/api/storage' | grep -q '/tmp/pocket-files-mock'"

check folder-create     bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'content-type: application/json' -d '{\"storage_id\":\"internal\",\"path\":\"Documents/SmokeFolder\"}' '$base/api/folders') = 201"
check operation-copy    bash -c "curl -fsS -X POST -H 'content-type: application/json' -d '{\"storage_id\":\"internal\",\"source\":\"Documents/Welcome.txt\",\"destination\":\"Documents/SmokeCopy.txt\"}' '$base/api/operations/copy' | grep -q '\"status\":\"ok\"'"
check operation-rename  bash -c "curl -fsS -X PUT -H 'content-type: application/json' -d '{\"name\":\"SmokeRenamed.txt\"}' '$base/api/files/internal%3ADocuments%2FSmokeCopy.txt/rename' | grep -q '\"destination\"'"
check delete-journal    bash -c "curl -fsS -X POST -H 'content-type: application/json' -d '{\"storage_id\":\"internal\",\"source\":\"Documents/SmokeRenamed.txt\"}' '$base/api/operations/delete' | grep -q '\"trash_path\"'"
check bulk-confirm      bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'content-type: application/json' -d '{\"action\":\"delete\",\"storage_id\":\"internal\",\"sources\":[\"Documents/SmokeFolder\"]}' '$base/api/operations/bulk') = 400"
check bulk-delete       bash -c "curl -fsS -X POST -H 'content-type: application/json' -d '{\"action\":\"delete\",\"confirm\":true,\"storage_id\":\"internal\",\"sources\":[\"Documents/SmokeFolder\"]}' '$base/api/operations/bulk' | grep -q '\"trash_path\"'"

check transfer-queue    bash -c "curl -fsS -X POST -H 'content-type: application/json' -d '{\"type\":\"copy\",\"storage_id\":\"internal\",\"conflict_strategy\":\"keep_both\",\"destination_storage\":\"usb\",\"items\":[{\"src\":\"Documents/Notes.txt\",\"dst\":\"Documents/SmokeCross.txt\"}]}' '$base/api/transfers' | grep -q '\"queued\"'"
check cross-device-file bash -c "sleep 1; curl -fsS '$base/api/files?storage_id=usb&path=Documents' | grep -q 'SmokeCross.txt'"
check transfer-list     bash -c "curl -fsS '$base/api/transfers' | grep -q '\"type\":\"copy\"'"
check transfer-order    bash -c "id=\$(curl -fsS '$base/api/transfers' | sed -E 's/.*\"id\":\"([^\"]+)\".*/\1/'); test \$(curl -s -o /dev/null -w '%{http_code}' -X PUT -H 'content-type: application/json' -d \"[\\\"\$id\\\"]\" '$base/api/transfers/order') = 200"

check favorites-add     bash -c "curl -fsS -X POST -H 'content-type: application/json' -d '{\"storage_id\":\"internal\",\"uri\":\"Documents/Welcome.txt\"}' '$base/api/favorites' >/dev/null"
check favorites-list    bash -c "curl -fsS '$base/api/favorites' | grep -q 'Documents/Welcome.txt'"
check tags-add          bash -c "curl -fsS -X POST -H 'content-type: application/json' -d '{\"storage_id\":\"internal\",\"uri\":\"Documents/Welcome.txt\",\"tag\":\"smoke\"}' '$base/api/tags' >/dev/null"
check tags-list         bash -c "curl -fsS '$base/api/tags' | grep -q 'smoke'"
check settings-roundtrip bash -c "curl -fsS -X PUT -H 'content-type: application/json' -d '{\"smoke\":\"1\"}' '$base/api/settings' >/dev/null; curl -fsS '$base/api/settings' | grep -q 'smoke'"
check recent-list       bash -c "curl -fsS '$base/api/recents' >/dev/null"

check views-books       bash -c "curl -fsS '$base/api/views/books' | grep -q 'Field Notes.pdf'"
check views-games       bash -c "curl -fsS '$base/api/views/games' | grep -q 'gba'"
check views-downloads   bash -c "curl -fsS '$base/api/views/downloads' | grep -q 'Example Archive.zip'"
check analytics         bash -c "curl -fsS '$base/api/analytics?storage_id=internal' | grep -q 'category_counts'"
check analytics-mock    bash -c "curl -fsS '$base/api/analytics?storage_id=internal' | grep -q '\"mock\":true'"
check downloads-state   bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' '$base/api/downloads') = 503"
check apps-unavailable  bash -c "curl -fsS '$base/api/apps/open?storage_id=internal&path=Documents/Welcome.txt' | grep -q 'unavailable'"
check providers-caps    bash -c "curl -fsS '$base/api/providers' | grep -q 'webdav'"
check unimplemented-501 bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' -X POST '$base/api/storage/internal/mount') = 501"
check unknown-route     bash -c "test \$(curl -s -o /dev/null -w '%{http_code}' '$base/api/nope') = 404"

if [ "$fail" -ne 0 ]; then echo "API smoke failed"; exit 1; fi
echo "API smoke passed against $base"
