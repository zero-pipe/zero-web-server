#!/usr/bin/env bash
# zero-web-server dev orchestrator (Linux / macOS)
# Usage:
#   ./tools/dev.sh                  # backend + frontend (local MySQL/Redis if already running)
#   ./tools/dev.sh start --docker   # optional: docker compose up MySQL/Redis
#   ./tools/dev.sh start --media
#   ./tools/dev.sh check            # probe MySQL/Redis/Docker
#   ./tools/dev.sh stop | status | restart

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEV_DIR="$ROOT/.dev"
STATE_FILE="$DEV_DIR/state.json"
LOG_DIR="$DEV_DIR/logs"
BACKEND_PORT=18080
FRONTEND_PORT=9528
CONFIG="${CONFIG:-configs/config.yaml}"

ACTION="${1:-start}"
shift || true

DOCKER=0
MEDIA=0
DETACH=0
NO_BROWSER=0
QUIET=0
REQUIRE_DEPS=0
SKIP_DEPS=0
SKIP_BUILD=0
while [[ $# -gt 0 ]]; do
    case "$1" in
        --docker) DOCKER=1 ;;
        --media) MEDIA=1 ;;
        --detach) DETACH=1 ;;
        --no-browser) NO_BROWSER=1 ;;
        --quiet) QUIET=1 ;;
        --require-deps) REQUIRE_DEPS=1 ;;
        --skip-deps-check) SKIP_DEPS=1 ;;
        --skip-build) SKIP_BUILD=1 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
    shift
done

declare -A LOG_OFFSETS=()

init_log_offsets_to_end() {
    LOG_OFFSETS=()
    local f size
    for f in "$LOG_DIR"/*.log; do
        [[ -f "$f" ]] || continue
        size=$(wc -c <"$f" | tr -d ' ')
        LOG_OFFSETS[$f]=$size
    done
}

log_line_noise() {
    [[ "$1" == *"[webpack.Progress]"* ]] && return 0
    [[ "$1" =~ ^[[:space:]]*INFO[[:space:]]+Starting\ development\ server ]] && return 0
    [[ "$1" == *"To create a production build, run npm run build"* ]] && return 0
    [[ "$1" == *"DeprecationWarning"* && "$1" == *"util._extend"* ]] && return 0
    return 1
}

follow_new_logs() {
    local f tag off size
    for f in "$LOG_DIR"/*.log; do
        [[ -f "$f" ]] || continue
        tag=$(basename "$f")
        tag=${tag%.out.log}; tag=${tag%.err.log}
        off=${LOG_OFFSETS[$f]:-0}
        size=$(wc -c <"$f" | tr -d ' ')
        [[ "$size" -gt "$off" ]] || continue
        tail -c +"$((off + 1))" "$f" | while IFS= read -r line || [[ -n "$line" ]]; do
            [[ "$line" =~ [[:space:]]*$ ]] && continue
            log_line_noise "$line" && continue
            printf '[%s] %s\n' "$tag" "$line"
        done
        LOG_OFFSETS[$f]=$size
    done
}

cyan() { printf '\033[36m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }
red() { printf '\033[31m%s\033[0m\n' "$*"; }

ensure_dirs() { mkdir -p "$DEV_DIR" "$LOG_DIR"; }

port_open() {
    (echo >/dev/tcp/127.0.0.1/"$1") 2>/dev/null
}

wait_port() {
    local port=$1 timeout=${2:-90} i=0
    while [[ $i -lt $timeout ]]; do
        port_open "$port" && return 0
        sleep 1
        i=$((i + 1))
    done
    return 1
}

docker_available() {
    command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

show_deps_hints() {
    echo ""
    yellow "  MySQL/Redis not ready on localhost?"
    echo "    A) Local install — start MySQL (:3306) + Redis (:6379), edit configs/config.yaml"
    echo "    B) Have Docker —  ./tools/dev.sh start --docker"
    echo "    C) Check only —  ./tools/dev.sh check"
}

check_local_deps() {
    local quiet=${1:-0} mysql=0 redis=0
    port_open 3306 && mysql=1
    port_open 6379 && redis=1
    if [[ "$quiet" -eq 0 ]]; then
        cyan "== Dependencies (MySQL / Redis) =="
        [[ "$mysql" -eq 1 ]] && green "  MySQL :3306  OK" || yellow "  MySQL :3306  not reachable"
        [[ "$redis" -eq 1 ]] && green "  Redis :6379  OK" || yellow "  Redis :6379  not reachable"
    fi
    [[ "$mysql" -eq 1 && "$redis" -eq 1 ]]
}

ensure_local_deps() {
    [[ "$SKIP_DEPS" -eq 1 ]] && return 0
    if check_local_deps 0; then return 0; fi
    show_deps_hints
    if [[ "$REQUIRE_DEPS" -eq 1 ]]; then
        red "MySQL/Redis required (--require-deps)."
        exit 1
    fi
    echo ""
    echo "  Continuing without deps (backend may fail until DB is up)..."
    if [[ "$DETACH" -eq 0 && -t 0 ]]; then
        read -r -p "Continue? [y/N]: " ans
        [[ "$ans" =~ ^[yY] ]] || exit 0
    fi
}

stop_pid() {
    local pid=$1 label=$2
    [[ "$pid" -gt 0 ]] 2>/dev/null || return 0
    if kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
        pkill -P "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
        echo "  stopped $label (pid $pid)"
    fi
}

read_state_pid() {
    local key=$1
    grep -o "\"$key\":[0-9]*" "$STATE_FILE" 2>/dev/null | head -1 | cut -d: -f2 || echo 0
}

stop_port_listeners() {
    local port=$1
    if command -v fuser >/dev/null 2>&1; then
        fuser -k "${port}/tcp" 2>/dev/null && echo "  stopped port:$port (fuser)" || true
        return
    fi
    if command -v lsof >/dev/null 2>&1; then
        local pids
        pids=$(lsof -ti ":$port" 2>/dev/null || true)
        if [[ -n "$pids" ]]; then
            echo "$pids" | xargs -r kill -9 2>/dev/null || true
            echo "  stopped port:$port (lsof)"
        fi
    fi
}

stop_all() {
    if [[ -f "$STATE_FILE" ]]; then
        stop_pid "$(read_state_pid media)" media
        stop_pid "$(read_state_pid frontend)" frontend
        stop_pid "$(read_state_pid backend)" backend
    fi
    stop_port_listeners 18080
    stop_port_listeners 9528
    if [[ -f "$STATE_FILE" ]] && [[ "$(read_state_pid media)" -gt 0 ]]; then
        stop_port_listeners 8080
    fi
    rm -f "$STATE_FILE"
}

ensure_config() {
    local cfg="$ROOT/$CONFIG"
    if [[ -f "$cfg" ]]; then return; fi
    echo "Missing $CONFIG — create configs/config.yaml (optional: configs/config.local.yaml for secrets)" >&2
    exit 1
}

ensure_frontend() {
    [[ -d "$ROOT/web/node_modules" ]] || { cyan "== npm install =="; (cd "$ROOT/web" && npm install); }
}

ensure_backend_built() {
    local exe="$ROOT/bin/zero-web-server"
    [[ "$SKIP_BUILD" -eq 1 ]] && { [[ -x "$exe" ]] || exe="$ROOT/bin/zero-web-server.exe"; echo "$exe"; return; }
    mkdir -p "$ROOT/bin"
    cyan "== Building backend (go build) =="
    (cd "$ROOT" && go build -o "$exe" ./cmd/server) || { red "go build failed"; exit 1; }
    green "Backend built: $exe"
    echo "$exe"
}

start_docker() {
    if ! docker_available; then
        red "Docker is not available (not installed or daemon not running)."
        echo "  - Use local MySQL/Redis:  ./tools/dev.sh start"
        echo "  - Or install/start Docker, then:  ./tools/dev.sh start --docker"
        exit 1
    fi
    cyan "== Docker: MySQL + Redis =="
    (cd "$ROOT/docker" && docker compose up -d)
    echo "Waiting for MySQL :3306 ..."
    if wait_port 3306 120; then green "MySQL ready"; else yellow "MySQL not ready — check docker logs"; fi
    if wait_port 6379 30; then green "Redis ready"; else yellow "Redis not ready"; fi
}

start_bg() {
    local tag=$1; shift
    local out="$LOG_DIR/$tag.out.log" err="$LOG_DIR/$tag.err.log"
    : >"$out"; : >"$err"
    "$@" >>"$out" 2>>"$err" &
    echo $!
}

find_media() {
    local zms="$(dirname "$ROOT")/zms"
    for p in "$zms/build/examples/demo_media_server" "$zms/build/examples/Release/demo_media_server"; do
        [[ -x "$p" ]] && { echo "$p|$zms"; return 0; }
    done
    return 1
}

save_state() {
    local b=$1 f=$2 m=${3:-0}
    ensure_dirs
    printf '{"backend":%s,"frontend":%s,"media":%s,"started":"%s"}\n' \
        "$b" "$f" "$m" "$(date -Iseconds 2>/dev/null || date)" >"$STATE_FILE"
}

do_check() {
    check_local_deps 0 || true
    echo ""
    if docker_available; then
        green "  Docker: available (use --docker to start MySQL/Redis containers)"
    else
        yellow "  Docker: not available — use local MySQL/Redis, or install Docker"
    fi
    if ! check_local_deps 1; then show_deps_hints; fi
}

do_status() {
    cyan "== zero-web-server dev status =="
    if [[ ! -f "$STATE_FILE" ]]; then
        echo "  Not started via dev.sh"
    else
        for row in "backend:$BACKEND_PORT" "frontend:$FRONTEND_PORT" "media:8080"; do
            name="${row%%:*}"; port="${row##*:}"
            pid="$(read_state_pid "$name")"
            [[ "$pid" -gt 0 ]] 2>/dev/null || continue
            if kill -0 "$pid" 2>/dev/null; then s=running; else s=stopped; fi
            printf "  %-10s pid=%-6s %-8s :%s\n" "$name" "$pid" "$s" "$port"
        done
    fi
    echo ""
    echo "  http://localhost:$FRONTEND_PORT"
}

do_start() {
    if [[ -f "$STATE_FILE" ]]; then
        yellow "State file exists — run ./tools/dev.sh stop first, or use restart"
    fi
    stop_all
    ensure_dirs
    ensure_config
    ensure_frontend
    if [[ "$DOCKER" -eq 1 ]]; then
        start_docker
    else
        ensure_local_deps
    fi

    cyan "== Starting backend :$BACKEND_PORT =="
    local backend backend_exe
    backend_exe=$(ensure_backend_built)
    backend=$(start_bg backend bash -c "cd '$ROOT' && exec '$backend_exe' -config '$CONFIG'")
    if ! wait_port "$BACKEND_PORT" 60; then
        tail -20 "$LOG_DIR/backend.err.log" 2>/dev/null || true
        stop_pid "$backend" backend
        red "Backend failed — see .dev/logs/backend.err.log"
        exit 1
    fi
    green "Backend ready (pid $backend)"

    cyan "== Starting frontend :$FRONTEND_PORT =="
    local frontend
    frontend=$(start_bg frontend bash -c "cd '$ROOT/web' && BROWSER=none exec npm run dev")
    if ! wait_port "$FRONTEND_PORT" 120; then
        tail -20 "$LOG_DIR/frontend.err.log" 2>/dev/null || true
        stop_pid "$frontend" frontend
        stop_pid "$backend" backend
        red "Frontend failed — see .dev/logs/frontend.err.log"
        exit 1
    fi
    green "Frontend ready (pid $frontend)"

    local media=0
    if [[ "$MEDIA" -eq 1 ]]; then
        if info=$(find_media); then
            IFS='|' read -r exe zroot <<<"$info"
            local cfg="conf/config.ini"
            [[ -f "$zroot/conf/config.zero-web-server.ini" ]] && cfg="conf/config.zero-web-server.ini"
            cyan "== Starting zero-media-server :8080 =="
            media=$(start_bg media bash -c "cd '$zroot' && exec '$exe' --config '$cfg'")
            green "Media server (pid $media)"
        else
            yellow "demo_media_server not found under ../zms/build — skip --media"
        fi
    fi

    save_state "$backend" "$frontend" "$media"

    cyan "== Dev stack running =="
    echo "  UI:    http://localhost:$FRONTEND_PORT"
    echo "  API:   http://localhost:$BACKEND_PORT"
    echo "  Logs:  $LOG_DIR"
    echo "  Stop:  ./tools/dev.sh stop"
    [[ "$NO_BROWSER" -eq 0 ]] && command -v xdg-open >/dev/null && xdg-open "http://localhost:$FRONTEND_PORT" 2>/dev/null || true

    if [[ "$DETACH" -eq 1 ]]; then
        echo "Detached — processes run in background."
        return 0
    fi

    trap 'stop_all; exit 0' INT TERM
    if [[ "$QUIET" -eq 1 ]]; then
        echo "Quiet mode — logs in $LOG_DIR (Ctrl+C stops all)"
    else
        echo "Streaming new log lines only (Ctrl+C stops all). Use --quiet for silent watch."
        init_log_offsets_to_end
    fi
    while kill -0 "$backend" 2>/dev/null && kill -0 "$frontend" 2>/dev/null; do
        [[ "$QUIET" -eq 0 ]] && follow_new_logs
        sleep 1
    done
    yellow "A process exited"
    stop_all
}

cd "$ROOT"
case "$ACTION" in
    start) do_start ;;
    stop) cyan "== Stopping zero-web-server (:18080 :9528) =="; stop_all
        if port_open 18080 || port_open 9528; then
            yellow "Some ports still open — check manually"
        else
            green "Done."
        fi
        if port_open 8080; then
            yellow "  :8080 still up (zero-media-server) — video may work without web-kit"
        fi
        ;;
    status) do_status ;;
    restart) stop_all; do_start ;;
    check) do_check ;;
    *) echo "Usage: $0 {start|stop|status|restart|check} [--docker] [--media] [--detach]"; exit 1 ;;
esac
