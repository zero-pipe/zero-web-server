.PHONY: run build tidy frontend test docker-up docker-down frontend-install frontend-dev frontend-build dev dev-stop dev-status verify-local mock-plugin agent-build agent-mock

EXE := $(shell if [ "$$(go env GOOS)" = "windows" ]; then echo .exe; fi)
AGENT_PLUGIN_DIR ?= bin/agent-plugins

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

# 启动 mock IPC 插件（用于验证插件接入链路）
mock-plugin:
	go run ./tools/mock-ipc-plugin

# 构建 zero-agent 与本地 mock 插件目录，用于联调。
agent-build:
	go build -o bin/zero-agent$(EXE) ./cmd/zero-agent
	mkdir -p $(AGENT_PLUGIN_DIR)/mock-ipc
	go build -o $(AGENT_PLUGIN_DIR)/mock-ipc/mock-ipc-plugin$(EXE) ./tools/mock-ipc-plugin
	cp tools/mock-ipc-plugin/plugin.yaml tools/mock-ipc-plugin/schema.json $(AGENT_PLUGIN_DIR)/mock-ipc/

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
