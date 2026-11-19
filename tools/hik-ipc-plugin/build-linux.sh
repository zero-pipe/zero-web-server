#!/usr/bin/env bash
# Build the Hikvision HCNetSDK sidecar on Linux (CGO required).
#
# Usage:
#   ./tools/hik-ipc-plugin/build-linux.sh /path/to/HCNetSDK_linux64
#   HIK_SDK_DIR=/path/to/HCNetSDK_linux64 ./tools/hik-ipc-plugin/build-linux.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HERE="$ROOT/tools/hik-ipc-plugin"
SDK_DIR="${1:-${HIK_SDK_DIR:-}}"

if [ -z "$SDK_DIR" ]; then
  echo "usage: $0 /path/to/HCNetSDK_linux64" >&2
  exit 1
fi
if [ ! -d "$SDK_DIR/头文件" ] || [ ! -d "$SDK_DIR/库文件" ]; then
  echo "SDK dir must contain 头文件/ and 库文件/: $SDK_DIR" >&2
  exit 1
fi

mkdir -p "$HERE/sdk/include" "$HERE/sdk/lib"
cp -f "$SDK_DIR/头文件/"*.h "$HERE/sdk/include/"
cp -f "$SDK_DIR/库文件/"*.so* "$HERE/sdk/lib/" 2>/dev/null || true
cp -rf "$SDK_DIR/库文件/HCNetSDKCom" "$HERE/sdk/lib/"

cd "$ROOT"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -tags hik -trimpath -o bin/hik-ipc-plugin ./tools/hik-ipc-plugin

mkdir -p bin/hik-lib
cp -f "$HERE/sdk/lib/"*.so* bin/hik-lib/ 2>/dev/null || true
cp -rf "$HERE/sdk/lib/HCNetSDKCom" bin/hik-lib/

# zero-agent 插件目录
AGENT_PLUGIN="$ROOT/bin/agent-plugins/hik-ipc"
rm -rf "$AGENT_PLUGIN"
mkdir -p "$AGENT_PLUGIN/lib"
cp -f bin/hik-ipc-plugin "$AGENT_PLUGIN/"
cp -f "$HERE/plugin.yaml" "$HERE/schema.json" "$AGENT_PLUGIN/"
cp -f "$HERE/sdk/lib/"*.so* "$AGENT_PLUGIN/lib/" 2>/dev/null || true
cp -rf "$HERE/sdk/lib/HCNetSDKCom" "$AGENT_PLUGIN/lib/"

echo "built: $ROOT/bin/hik-ipc-plugin"
echo "runtime libs: $ROOT/bin/hik-lib"
echo "agent plugin: $AGENT_PLUGIN"
echo
echo "run example:"
echo "  ./bin/zero-agent -config configs/agent.yaml"
