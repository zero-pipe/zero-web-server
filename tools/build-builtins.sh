#!/usr/bin/env bash
# 构建 zero-agent 与内置插件到 bin/agent-plugins/<id>/（best-effort）。
#
# 内置插件：
#   - mock-ipc：始终构建（开发联调用）。
#   - hik-ipc ：仅在检测到 HCNetSDK + 64 位 MinGW 工具链时用 -tags hik + cgo 真编；
#               否则跳过并告警（不放入桩，避免“内置已启用但跑的是桩”）。
#
# 环境变量：
#   HIK_SDK_DIR   海康 SDK 目录（含 头文件/ 库文件/）。缺省自动在常见路径探测。
#   HIK_CC/HIK_CXX 指定 64 位 gcc/g++（缺省自动探测 msys64 mingw64 或 x86_64-w64-mingw32-*）。
#
# 用法：tools/build-builtins.sh
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin"
PLUGDIR="$BIN/agent-plugins"
GOOS="$(go env GOOS)"
EXE=""
[ "$GOOS" = "windows" ] && EXE=".exe"

say()  { printf '[builtins] %s\n' "$*"; }
warn() { printf '[builtins] WARN: %s\n' "$*" >&2; }

mkdir -p "$PLUGDIR"

say "build zero-agent"
if ! (cd "$ROOT" && go build -o "$BIN/zero-agent$EXE" ./cmd/zero-agent); then
  warn "zero-agent 构建失败"
  exit 1
fi

# ---- mock 插件（始终构建）----
build_mock() {
  say "build mock-ipc"
  local out="$PLUGDIR/mock-ipc"
  mkdir -p "$out"
  (cd "$ROOT" && go build -o "$out/mock-ipc-plugin$EXE" ./tools/mock-ipc-plugin) || { warn "mock 构建失败"; return 1; }
  cp -f "$ROOT/tools/mock-ipc-plugin/plugin.yaml" "$ROOT/tools/mock-ipc-plugin/schema.json" "$out/"
  say "  -> $out"
}

# ---- hik 插件（仅真编成功才放入）----
find_hik_sdk() {
  [ -n "${HIK_SDK_DIR:-}" ] && [ -d "$HIK_SDK_DIR/头文件" ] && { echo "$HIK_SDK_DIR"; return 0; }
  local cand
  for cand in \
    /e/13_zero/tools/hiksdk/*Win64_ZH \
    e:/13_zero/tools/hiksdk/*Win64_ZH \
    "$ROOT/../tools/hiksdk"/*Win64_ZH; do
    [ -d "$cand/头文件" ] && [ -d "$cand/库文件" ] && { echo "$cand"; return 0; }
  done
  return 1
}

pick_toolchain() {
  if [ -n "${HIK_CC:-}" ] && [ -n "${HIK_CXX:-}" ]; then
    echo "${HIK_CC}|${HIK_CXX}"; return 0
  fi
  # 优先 msys64 mingw64（64 位）
  if [ -x /c/msys64/mingw64/bin/x86_64-w64-mingw32-g++.exe ]; then
    echo "/c/msys64/mingw64/bin/x86_64-w64-mingw32-gcc|/c/msys64/mingw64/bin/x86_64-w64-mingw32-g++"; return 0
  fi
  if command -v x86_64-w64-mingw32-g++ >/dev/null 2>&1; then
    echo "x86_64-w64-mingw32-gcc|x86_64-w64-mingw32-g++"; return 0
  fi
  return 1
}

build_hik() {
  local sdk tc cc cxx
  if ! sdk="$(find_hik_sdk)"; then
    warn "未找到 HCNetSDK（设置 HIK_SDK_DIR 指向含 头文件/库文件 的目录），跳过 hik 内置插件"
    return 0
  fi
  if ! tc="$(pick_toolchain)"; then
    warn "未找到 64 位 MinGW 工具链（x86_64-w64-mingw32-g++），跳过 hik 内置插件"
    return 0
  fi
  cc="${tc%%|*}"; cxx="${tc##*|}"
  say "build hik-ipc (SDK=$sdk, CXX=$cxx)"
  # msys64 工具链需要其 bin 在 PATH（gendef/dlltool）
  local pathpre=""
  case "$cxx" in
    /c/msys64/mingw64/bin/*) pathpre="/c/msys64/mingw64/bin:" ;;
  esac
  if ! ( export PATH="${pathpre}$PATH" CC="$cc" CXX="$cxx"; \
         bash "$ROOT/tools/hik-ipc-plugin/build-windows.sh" "$sdk" ); then
    warn "hik 构建失败，跳过（不放入桩）"
    return 0
  fi
  say "  -> $PLUGDIR/hik-ipc"
}

build_mock
build_hik

say "done. 插件目录：$PLUGDIR"
