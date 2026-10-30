.PHONY: all build test test-backend test-frontend e2e dev screenshots clean

FRONTEND := apps/pocket-files/frontend
BACKEND := apps/pocket-files/backend/filesd

all: build

# Production build: frontend bundle plus the filesd binary.
build:
	cd $(FRONTEND) && bun install && bun run build
	cd $(BACKEND) && go build -o filesd .

# One command runs the full suite: Go unit/integration tests with the race
# detector, i18n key parity, TypeScript, the production bundle, and a live API
# smoke test against a freshly started daemon.
test: test-backend test-frontend e2e

test-backend:
	cd $(BACKEND) && go vet ./... && go test -race -count=1 ./...

test-frontend:
	cd $(FRONTEND) && bun run check:i18n && bun run check:profiles && bun run typecheck && bun run build

e2e:
	@cd $(BACKEND) && go build -o /tmp/pocket-filesd-e2e .
	@POCKET_FILES_DB=/tmp/pocket-files-e2e.db POCKET_FILES_ADDR=127.0.0.1:8788 /tmp/pocket-filesd-e2e & \
	  daemon=$$!; \
	  trap 'kill $$daemon 2>/dev/null' EXIT; \
	  for i in $$(seq 1 50); do \
	    if curl -fsS http://127.0.0.1:8788/api/health >/dev/null 2>&1; then break; fi; \
	    sleep 0.2; \
	  done; \
	  POCKET_FILES_URL=http://127.0.0.1:8788 bash apps/pocket-files/tests/smoke.sh

# Start filesd and the Vite dev server together; Ctrl-C stops both.
dev:
	@bash apps/pocket-files/tests/dev.sh

screenshots:
	node apps/pocket-files/tests/capture-screenshots.mjs http://127.0.0.1:5180 apps/pocket-files/docs/screenshots

clean:
	rm -rf $(FRONTEND)/dist $(BACKEND)/filesd /tmp/pocket-filesd-e2e
