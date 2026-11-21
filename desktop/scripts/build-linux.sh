#!/usr/bin/env bash
# Builds the Linux desktop app in an Ubuntu container.
#
#   desktop/scripts/build-linux.sh [amd64|arm64]
#
# The binary is written to desktop/build/bin/munna-proxy-linux-<arch>. Other
# architectures run under emulation and are much slower.
set -euo pipefail

ARCH="${1:-}"
if [[ -z "$ARCH" ]]; then
	case "$(uname -m)" in
		arm64|aarch64) ARCH=arm64 ;;
		x86_64) ARCH=amd64 ;;
		*) echo "unsupported host architecture" >&2; exit 1 ;;
	esac
fi
case "$ARCH" in
	amd64|arm64) ;;
	*) echo "usage: $0 [amd64|arm64]" >&2; exit 1 ;;
esac

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
IMAGE="munna-desktop-linux-build:${ARCH}"
OUT="desktop/build/bin/munna-proxy-linux-${ARCH}"

docker build --platform "linux/${ARCH}" -f "$ROOT/desktop/Dockerfile.linux" \
	--build-arg TARGETARCH="$ARCH" -t "$IMAGE" "$ROOT/desktop"

docker run --rm --platform "linux/${ARCH}" -v "$ROOT:/src" -v munna-gomod-cache:/root/go/pkg/mod -w /src/desktop "$IMAGE" \
	sh -c "wails build -platform linux/${ARCH} -tags webkit2_41 -skipbindings -o munna-proxy-linux-${ARCH} && chown -R $(id -u):$(id -g) /src/desktop/build/bin"

echo "built $OUT"
