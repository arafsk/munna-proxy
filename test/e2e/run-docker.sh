#!/usr/bin/env bash
# Two-device LAN test using Docker.
#
#   munna-host    runs `munna serve` on 0.0.0.0:1080 and a web server that only
#                 listens on its own 127.0.0.1 (the "localhost-only" service)
#   munna-client  is a separate device on the same bridge network with curl only
#
# Every check runs from the client, so the traffic crosses the container network
# exactly as it would cross a LAN. Requires Docker and Go.
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
NET="munna-lan-test"
HOST="munna-host"
CLIENT="munna-client"
IMAGE="munna-e2e:latest"
USER_NAME="alice"
PASSWORD="s3cret-pass"
PROXY_PORT=1080
SECRET_PORT=8000

pass=0
fail=0
ctx="$(mktemp -d)"

cleanup() {
	docker rm -f "$HOST" "$CLIENT" >/dev/null 2>&1
	docker network rm "$NET" >/dev/null 2>&1
	rm -rf "$ctx"
}
trap cleanup EXIT

check() {
	local name="$1" want="$2" got="$3"
	if [[ "$got" == "$want" ]]; then
		printf 'PASS  %-40s %s\n' "$name" "$got"
		pass=$((pass + 1))
	else
		printf 'FAIL  %-40s want=%s got=%s\n' "$name" "$want" "$got"
		fail=$((fail + 1))
	fi
}

# http_code runs curl inside the client container and prints the HTTP status.
# Failures print "000" (no response), so the caller can compare exact values.
http_code() {
	docker exec "$CLIENT" curl -s -o /dev/null -w '%{http_code}' --max-time 8 "$@" 2>/dev/null || true
}

echo "==> building linux binary for the container platform"
arch="$(docker version --format '{{.Server.Arch}}')"
case "$arch" in
	amd64|arm64) ;;
	x86_64) arch=amd64 ;;
	aarch64) arch=arm64 ;;
	*) echo "unsupported docker arch: $arch" >&2; exit 1 ;;
esac
(cd "$ROOT" && CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -o "$ctx/munna" ./cmd/munna) || exit 1
cp "$ROOT/test/e2e/Dockerfile" "$ctx/Dockerfile"

echo "==> building image $IMAGE"
docker build -q -t "$IMAGE" "$ctx" >/dev/null || exit 1

cleanup
docker network create "$NET" >/dev/null || exit 1

echo "==> starting host and client containers"
docker run -d --name "$HOST" --network "$NET" "$IMAGE" \
	"python3 -m http.server $SECRET_PORT --bind 127.0.0.1 >/dev/null 2>&1 & exec munna serve -listen 0.0.0.0:$PROXY_PORT -user $USER_NAME -password $PASSWORD -mdns=false" >/dev/null || exit 1
docker run -d --name "$CLIENT" --network "$NET" "$IMAGE" "sleep 36000" >/dev/null || exit 1
sleep 2

HOST_IP="$(docker inspect -f "{{(index .NetworkSettings.Networks \"$NET\").IPAddress}}" "$HOST")"
CLIENT_IP="$(docker inspect -f "{{(index .NetworkSettings.Networks \"$NET\").IPAddress}}" "$CLIENT")"
echo "host=$HOST_IP client=$CLIENT_IP"
echo

echo "==> host output"
docker logs "$HOST" 2>&1 | grep -v '^[█▄▀ ]*$' | head -12
echo

invite="$(docker logs "$HOST" 2>&1 | grep -o 'munna://[^ ]*' | head -1)"
check "invite advertises the LAN address" "1" "$( [[ "$invite" == *"@$HOST_IP:$PROXY_PORT"* ]] && echo 1 || echo 0 )"

echo "==> client checks"
proxy="socks5h://$USER_NAME:$PASSWORD@$HOST_IP:$PROXY_PORT"

check "reach host-loopback service via proxy" "200" "$(http_code --proxy "$proxy" "http://127.0.0.1:$SECRET_PORT/")"
check "same service directly from client" "000" "$(http_code "http://$HOST_IP:$SECRET_PORT/")"
check "wrong password rejected" "000" "$(http_code --proxy "socks5h://$USER_NAME:wrong@$HOST_IP:$PROXY_PORT" "http://127.0.0.1:$SECRET_PORT/")"
check "no credentials rejected" "000" "$(http_code --proxy "socks5h://$HOST_IP:$PROXY_PORT" "http://127.0.0.1:$SECRET_PORT/")"
check "cloud metadata endpoint denied" "000" "$(http_code --proxy "$proxy" "http://169.254.169.254/latest/meta-data/")"
check "public internet via proxy" "200" "$(http_code --proxy "$proxy" "https://example.com/")"

echo "==> lockout after repeated failures"
for _ in 1 2 3 4 5; do
	http_code --proxy "socks5h://$USER_NAME:wrong@$HOST_IP:$PROXY_PORT" "http://127.0.0.1:$SECRET_PORT/" >/dev/null
done
check "correct password while locked out" "000" "$(http_code --proxy "$proxy" "http://127.0.0.1:$SECRET_PORT/")"

echo
echo "passed=$pass failed=$fail"
[[ "$fail" -eq 0 ]]
