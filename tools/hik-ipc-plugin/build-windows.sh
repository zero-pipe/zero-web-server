#!/usr/bin/env bash
# Build the Hikvision HCNetSDK sidecar on Windows (Go + CGO + MinGW g++).
#
# Requirements:
#   - Go (amd64), MinGW-w64 gcc/g++ on PATH
#   - gendef + dlltool (MinGW binutils) on PATH
#
# Usage:
#   ./tools/hik-ipc-plugin/build-windows.sh /path/to/HCNetSDK_Win64_ZH
#   HIK_SDK_DIR=/path/to/HCNetSDK_Win64_ZH ./tools/hik-ipc-plugin/build-windows.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HERE="$ROOT/tools/hik-ipc-plugin"
SDK_DIR="${1:-${HIK_SDK_DIR:-}}"

if [ -z "$SDK_DIR" ]; then
  echo "usage: $0 /path/to/HCNetSDK_Win64_ZH" >&2
  exit 1
fi
if [ ! -d "$SDK_DIR/头文件" ] || [ ! -d "$SDK_DIR/库文件" ]; then
  echo "SDK dir must contain 头文件/ and 库文件/: $SDK_DIR" >&2
  exit 1
fi

mkdir -p "$HERE/sdk/include" "$HERE/sdk/lib"
cp -f "$SDK_DIR/头文件/"*.h "$HERE/sdk/include/"
cp -f "$SDK_DIR/库文件/"*.dll "$HERE/sdk/lib/"
cp -f "$SDK_DIR/库文件/"*.lib "$HERE/sdk/lib/" 2>/dev/null || true
rm -rf "$HERE/sdk/lib/HCNetSDKCom"
cp -rf "$SDK_DIR/库文件/HCNetSDKCom" "$HERE/sdk/lib/"

# MinGW links MinGW-style import libs; generate them from the official DLLs.
pushd "$HERE/sdk/lib" >/dev/null
gendef -a HCNetSDK.dll > HCNetSDK.def 2>/dev/null || gendef HCNetSDK.dll > HCNetSDK.def
gendef -a HCCore.dll > HCCore.def 2>/dev/null || gendef HCCore.dll > HCCore.def
dlltool -d HCNetSDK.def -l libHCNetSDK.a -D HCNetSDK.dll
dlltool -d HCCore.def -l libHCCore.a -D HCCore.dll
popd >/dev/null

cd "$ROOT"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  go build -tags hik -trimpath -o bin/hik-ipc-plugin.exe ./tools/hik-ipc-plugin

# Runtime bundle: exe + all DLLs + HCNetSDKCom in one directory.
OUT="$ROOT/bin/hik-win"
rm -rf "$OUT"
mkdir -p "$OUT"
cp -f bin/hik-ipc-plugin.exe "$OUT/"
cp -f "$SDK_DIR/库文件/"*.dll "$OUT/"
cp -rf "$SDK_DIR/库文件/HCNetSDKCom" "$OUT/"

# zero-agent 插件目录
AGENT_PLUGIN="$ROOT/bin/agent-plugins/hik-ipc"
rm -rf "$AGENT_PLUGIN"
mkdir -p "$AGENT_PLUGIN"
cp -f bin/hik-ipc-plugin.exe "$OUT/"*.dll "$AGENT_PLUGIN/" 2>/dev/null || true
cp -rf "$OUT/HCNetSDKCom" "$AGENT_PLUGIN/"
cp -f "$HERE/plugin.yaml" "$HERE/schema.json" "$AGENT_PLUGIN/"

echo "built: $OUT/hik-ipc-plugin.exe"
echo "agent plugin: $AGENT_PLUGIN"
echo
echo "run example:"
echo "  ./bin/zero-agent -config configs/agent.yaml"
