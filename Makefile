.PHONY: build test dev
build:
	cd apps/pocket-files/frontend && bun install --frozen-lockfile=false && bun run build
	cd apps/pocket-files/backend/filesd && go build ./...
test:
	cd apps/pocket-files/backend/filesd && go test ./...
dev:
	@trap 'kill 0' EXIT; cd apps/pocket-files/backend/filesd && go run . & cd apps/pocket-files/frontend && bun install --frozen-lockfile=false && bun run dev
