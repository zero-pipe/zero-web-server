.PHONY: run build tidy frontend test docker-up docker-down frontend-install frontend-dev frontend-build

run:
	go run ./cmd/server -config configs/config.yaml

build:
	go build -o bin/zero-web-kit ./cmd/server

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
