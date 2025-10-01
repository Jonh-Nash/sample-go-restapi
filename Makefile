.PHONY: serve seed dev

PORT ?= 8080
API_BASE_URL ?= http://localhost:$(PORT)

GO := /opt/homebrew/bin/go
GOENV := PATH=/opt/homebrew/bin:$$PATH
GO_RUN := $(GOENV) $(GO) run

serve:
	PORT=$(PORT) $(GO_RUN) ./cmd/api-server

seed:
	API_BASE_URL=$(API_BASE_URL) $(GO_RUN) ./cmd/seed

dev:
	@set -euo pipefail; \
	PORT=$(PORT) $(GO_RUN) ./cmd/api-server & \
	server_pid=$$!; \
	trap 'kill $$server_pid >/dev/null 2>&1' EXIT; \
	API_BASE_URL=$(API_BASE_URL) $(GO_RUN) ./cmd/seed; \
	wait $$server_pid
