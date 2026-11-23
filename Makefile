.PHONY: run build tidy frontend test docker-up docker-down frontend-install frontend-dev frontend-build dev dev-stop dev-status verify-local agent-build agent-run

EXE := $(shell if [ "$$(go env GOOS)" = "windows" ]; then echo .exe; fi)
AGENT_PLUGIN_DIR ?= bin/agent-plugins

run:
	go run ./cmd/server -config configs/config.yaml

# One-shot dev: backend + frontend (local MySQL/Redis; add --docker if using Docker)
dev:
	@if [ -x tools/dev.sh ]; then ./tools/dev.sh start; \
	else echo "Run: tools\\dev.cmd start (Windows) or powershell -ExecutionPolicy Bypass -File tools/dev.ps1 start"; fi

dev-stop:
	@if [ -x tools/dev.sh ]; then ./tools/dev.sh stop; \
	else echo "Run: tools\\dev.cmd stop (Windows) or powershell -ExecutionPolicy Bypass -File tools/dev.ps1 stop"; fi

dev-status:
	@if [ -x tools/dev.sh ]; then ./tools/dev.sh status; \
	else echo "Run: tools\\dev.cmd status (Windows) or powershell -ExecutionPolicy Bypass -File tools/dev.ps1 status"; fi

build:
	go build -o bin/zero-web-server ./cmd/server

tidy:
	go mod tidy

test:
	go test ./...

# 构建 zero-agent + 内置插件（hik 仅在 SDK+64位工具链时真编）到 bin/agent-plugins/。
agent-build:
	bash tools/build-builtins.sh

agent-run:
	go run ./cmd/zero-agent -config configs/agent.yaml

# 本机 Go 版本低于 go.mod 锁定的工具链时，临时降级 go 指令做本地 build/test，结束后自动恢复。
# 仅用于本地验证，不要提交降级后的 go.mod。
verify-local:
	@cp go.mod go.mod.bak && \
	sed -i 's/^go 1\.25\.4$$/go 1.25.3/' go.mod && \
	{ GOTOOLCHAIN=local go build ./... && GOTOOLCHAIN=local go test ./... ; } ; \
	status=$$? ; mv go.mod.bak go.mod ; exit $$status

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
