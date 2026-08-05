.PHONY: run build tidy frontend test docker-up docker-down frontend-install frontend-dev frontend-build dev dev-stop dev-status

run:
	go run ./cmd/server -config configs/config.yaml

# One-shot dev: backend + frontend (local MySQL/Redis; add --docker if using Docker)
dev:
	@if [ -x tools/dev.sh ]; then ./tools/dev.sh start; \
	else echo "Run: powershell -File tools/dev.ps1 start"; fi

dev-stop:
	@if [ -x tools/dev.sh ]; then ./tools/dev.sh stop; \
	else echo "Run: powershell -File tools/dev.ps1 stop"; fi

dev-status:
	@if [ -x tools/dev.sh ]; then ./tools/dev.sh status; \
	else echo "Run: powershell -File tools/dev.ps1 status"; fi

build:
	go build -o bin/zero-web-server ./cmd/server

tidy:
	go mod tidy

test:
	go test ./...

docker-up:
	cd docker && docker compose up -d

docker-down:
	cd docker && docker compose down

frontend-install:
	cd web && npm install

frontend-dev:
	cd web && npm run dev

frontend-build:
	cd web && npm run build
