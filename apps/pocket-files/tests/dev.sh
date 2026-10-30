#!/usr/bin/env bash
# Start filesd and the Vite dev server together; Ctrl-C stops both.
#
# Ports are fixed (8787 for the API, 5180 for the UI) so other PJOS apps that
# default to 5173 never collide with Pocket Files during development.
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
backend="$repo_root/apps/pocket-files/backend/filesd"
frontend="$repo_root/apps/pocket-files/frontend"

api_port=${POCKET_FILES_ADDR:-127.0.0.1:8787}
ui_port=${POCKET_FILES_UI_PORT:-5180}

pids=()
cleanup() {
  for pid in "${pids[@]:-}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "filesd: http://$api_port   ui: http://127.0.0.1:$ui_port"
(cd "$backend" && POCKET_FILES_ADDR="$api_port" exec go run .) &
pids+=($!)

(cd "$frontend" && bun install >/dev/null && exec bunx vite --host 127.0.0.1 --port "$ui_port" --strictPort) &
pids+=($!)

wait -n "${pids[@]}"
