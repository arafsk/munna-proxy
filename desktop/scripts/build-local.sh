#!/usr/bin/env bash
# Builds the macOS and Windows desktop apps on this machine with Wails.
# Linux is built in a container by build-linux.sh.
#
#   desktop/scripts/build-local.sh
#
# Outputs go to desktop/build/bin/ with platform names so artifacts do not
# overwrite each other.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT/desktop"

WAILS="$(go env GOPATH)/bin/wails"
if [[ ! -x "$WAILS" ]]; then
	echo "wails CLI not found; install with: go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1" >&2
	exit 1
fi

echo "==> macOS universal"
rm -rf "build/bin/MunnaProxy-macos-universal.app"
"$WAILS" build -platform darwin/universal -skipbindings
mv "build/bin/Munna Proxy.app" "build/bin/MunnaProxy-macos-universal.app"

echo "==> Windows amd64"
"$WAILS" build -platform windows/amd64 -skipbindings -o munna-proxy-windows-amd64.exe

echo "==> Windows arm64"
"$WAILS" build -platform windows/arm64 -skipbindings -o munna-proxy-windows-arm64.exe

ls -la build/bin
