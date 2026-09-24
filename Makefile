# Settle monorepo helpers
.PHONY: help setup infra-up infra-down frontend-install frontend-dev backend-tidy backend-run backend-build

help:
	@echo "Settle commands:"
	@echo "  make setup            - install frontend deps + tidy Go modules"
	@echo "  make infra-up         - start Postgres + MinIO"
	@echo "  make infra-down       - stop infra"
	@echo "  make frontend-dev     - Next.js dev server"
	@echo "  make backend-run      - run Go API stub"
	@echo "  make backend-build    - build Go API binary"

setup: frontend-install backend-tidy

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

backend-tidy:
	cd backend && go mod tidy

backend-run:
	cd backend && go run ./cmd/settle

backend-build:
	cd backend && go build -o bin/settle ./cmd/settle

infra-up:
	docker compose up -d

infra-down:
	docker compose down
