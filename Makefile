.PHONY: help dev-deps frontend-install frontend-dev frontend-build frontend-lint \
	backend-run backend-test backend-build test build compose-up compose-down compose-build lint

help:
	@echo "Cibles Vibed :"
	@echo "  make frontend-install  — npm ci dans frontend/"
	@echo "  make frontend-dev      — Vite (proxy /api → :8080)"
	@echo "  make frontend-build    — build production du front"
	@echo "  make backend-run       — lance le serveur Go (nécessite DATABASE_URL / .env)"
	@echo "  make backend-test      — go test ./..."
	@echo "  make backend-build     — go build"
	@echo "  make test              — tests backend + lint front"
	@echo "  make build             — build front + backend"
	@echo "  make compose-up        — docker compose up --build"
	@echo "  make compose-down      — docker compose down"

frontend-install:
	cd frontend && npm ci

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

frontend-lint:
	cd frontend && npm run lint

backend-run:
	cd backend && STATIC_DIR=$${STATIC_DIR:-../frontend/dist} go run ./cmd/server


backend-test:
	cd backend && go test ./...

backend-build:
	cd backend && go build -o bin/server ./cmd/server

lint: frontend-lint
	cd backend && go vet ./...

test: backend-test frontend-lint

build: frontend-build backend-build

compose-up:
	docker compose up --build

compose-build:
	docker compose build

compose-down:
	docker compose down
