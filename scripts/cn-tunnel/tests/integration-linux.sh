#!/usr/bin/env bash
# Run only on an ephemeral Ubuntu CI runner, as root.
set -Eeuo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
SERVER_PID=""
cleanup() {
  [ -z "$SERVER_PID" ] || kill "$SERVER_PID" 2>/dev/null || true
  systemctl disable --now wg-quick@wg-newapi 2>/dev/null || true
  ip netns del cn-test 2>/dev/null || true
  ip link del cn-veth 2>/dev/null || true
  rm -f /etc/wireguard/wg-newapi.conf /usr/local/share/ca-certificates/cn-tunnel-test.crt
  update-ca-certificates >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
[ "$(id -u)" -eq 0 ]
[ ! -e /etc/wireguard/wg-newapi.conf ]
trap cleanup EXIT
apt-get update -qq
apt-get install -y wireguard-tools iproute2 openssl curl zip iptables >/dev/null
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -keyout "$WORK/key.pem" \
  -out "$WORK/cert.pem" -subj '/CN=cn.meta-api.vip' \
  -addext 'subjectAltName=DNS:cn.meta-api.vip' >/dev/null 2>&1
cp "$WORK/cert.pem" /usr/local/share/ca-certificates/cn-tunnel-test.crt
update-ca-certificates >/dev/null
cat > "$WORK/https.py" <<'PY'
import http.server, ssl, sys, time
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        if self.path == '/v1/stream-test':
            for data in (b'data: {"delta":"hello"}\n\n', b'data: [DONE]\n\n'):
                self.wfile.write(data)
                self.wfile.flush()
                time.sleep(.2)
        else:
            self.wfile.write(b'{"success":true}')
server = http.server.ThreadingHTTPServer(('0.0.0.0', 443), Handler)
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
ctx.load_cert_chain(sys.argv[1], sys.argv[2])
server.socket = ctx.wrap_socket(server.socket, server_side=True)
server.serve_forever()
PY
python3 "$WORK/https.py" "$WORK/cert.pem" "$WORK/key.pem" >"$WORK/https.log" 2>&1 &
SERVER_PID=$!
for i in $(seq 1 20); do
  if curl --noproxy '*' -fsS --resolve cn.meta-api.vip:443:127.0.0.1 https://cn.meta-api.vip/api/status >/dev/null 2>&1; then break; fi
  sleep .2
done
bash "$ROOT/server-install.sh" --endpoint 192.0.2.1 --client-name ci-client --output-dir "$WORK/bundles"
ip netns add cn-test
ip link add cn-veth type veth peer name cn-peer
ip link set cn-peer netns cn-test
ip addr add 192.0.2.1/24 dev cn-veth
ip link set cn-veth up
ip netns exec cn-test ip addr add 192.0.2.2/24 dev cn-peer
ip netns exec cn-test ip link set cn-peer up
ip netns exec cn-test ip link set lo up
ip netns exec cn-test wg-quick up "$WORK/bundles/ci-client/ci-client.conf"
# No default route in the namespace: HTTPS can only travel through WireGuard.
ip netns exec cn-test curl --noproxy '*' -fsS --connect-timeout 5 --max-time 15 \
  --resolve cn.meta-api.vip:443:10.66.0.1 https://cn.meta-api.vip/api/status > "$WORK/status"
rg_or_grep() { grep -F "$1" "$2"; }
rg_or_grep '"success":true' "$WORK/status"
ip netns exec cn-test curl --noproxy '*' -NfsS --max-time 15 \
  --resolve cn.meta-api.vip:443:10.66.0.1 https://cn.meta-api.vip/v1/stream-test > "$WORK/stream"
rg_or_grep 'data: [DONE]' "$WORK/stream"
wg show wg-newapi latest-handshakes | awk '$2 > 0 { ok=1 } END { exit !ok }'
# Confirm peer additions do not interrupt the first client's access.
bash "$ROOT/server-install.sh" --endpoint 192.0.2.1 --add-client --client-name ci-second --output-dir "$WORK/bundles"
ip netns exec cn-test curl --noproxy '*' -fsS --max-time 10 \
  --resolve cn.meta-api.vip:443:10.66.0.1 https://cn.meta-api.vip/api/status >/dev/null
printf 'WireGuard HTTPS/API/SSE integration passed\n'
