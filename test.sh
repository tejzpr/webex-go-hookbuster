#!/bin/bash
# ──────────────────────────────────────────────────────────────────────────
# Hookbuster Test Script
#
# Starts a local HTTP receiver, then launches hookbuster in deployment
# mode (firehose) so you can trigger Webex events and see them forwarded.
#
# Usage:
#   ./test.sh                          # prompts for token
#   WEBEX_TOKEN=<token> ./test.sh      # uses env var
# ──────────────────────────────────────────────────────────────────────────

set -e

RECV_PORT=8080
TARGET="localhost"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="$SCRIPT_DIR/hookbuster"
RECV_PID=""
HOOKBUSTER_PID=""

# ── Colors ───────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

info()  { echo -e "${BLUE}[INFO]${NC}  $1"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $1"; }
err()   { echo -e "${RED}[ERROR]${NC} $1"; }

# ── Cleanup on exit ─────────────────────────────────────────────────────
cleanup() {
    echo ""
    info "Shutting down..."

    if [ -n "$HOOKBUSTER_PID" ] && kill -0 "$HOOKBUSTER_PID" 2>/dev/null; then
        kill -INT "$HOOKBUSTER_PID" 2>/dev/null
        wait "$HOOKBUSTER_PID" 2>/dev/null || true
        ok "Hookbuster stopped"
    fi

    if [ -n "$RECV_PID" ] && kill -0 "$RECV_PID" 2>/dev/null; then
        kill "$RECV_PID" 2>/dev/null
        wait "$RECV_PID" 2>/dev/null || true
        ok "Receiver stopped"
    fi

    ok "Done."
}
trap cleanup EXIT INT TERM

# ── Step 1: Build ────────────────────────────────────────────────────────
info "Building hookbuster..."
cd "$SCRIPT_DIR"
go build -o "$BINARY" .
ok "Built: $BINARY"

# ── Step 2: Get Webex token ──────────────────────────────────────────────
TOKEN="${WEBEX_TOKEN:-}"
if [ -z "$TOKEN" ]; then
    echo ""
    echo -e "${BOLD}Enter your Webex access token${NC}"
    echo "  (Get one at https://developer.webex.com)"
    echo -n "> "
    read -r TOKEN
    echo ""
fi

if [ -z "$TOKEN" ]; then
    err "No token provided. Exiting."
    exit 1
fi

# ── Step 3: Start local HTTP receiver ────────────────────────────────────
info "Starting local HTTP receiver on port $RECV_PORT..."

python3 -u -c "
from http.server import HTTPServer, BaseHTTPRequestHandler
import json, sys

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length)
        try:
            data = json.loads(body)
            resource = data.get('resource', '?')
            event = data.get('event', '?')
            print(f'')
            print(f'========== RECEIVED: {resource}:{event} ==========')
            print(json.dumps(data, indent=2))
            print(f'=================================================')
            print(f'')
            sys.stdout.flush()
        except Exception as e:
            print(f'[receiver] parse error: {e}')
            sys.stdout.flush()
        self.send_response(200)
        self.end_headers()

    def log_message(self, format, *args):
        pass  # suppress default access logs

print(f'[receiver] Listening on port $RECV_PORT ...')
sys.stdout.flush()
HTTPServer(('', $RECV_PORT), Handler).serve_forever()
" &
RECV_PID=$!
sleep 1

if ! kill -0 "$RECV_PID" 2>/dev/null; then
    err "Failed to start receiver. Is port $RECV_PORT already in use?"
    exit 1
fi
ok "Receiver running (PID $RECV_PID)"

# ── Step 4: Start hookbuster in deployment (firehose) mode ───────────────
info "Starting hookbuster (firehose mode → all resources, all events)..."
echo ""

TOKEN="$TOKEN" PORT="$RECV_PORT" TARGET="$TARGET" "$BINARY" &
HOOKBUSTER_PID=$!
sleep 3

if ! kill -0 "$HOOKBUSTER_PID" 2>/dev/null; then
    err "Hookbuster exited unexpectedly. Check the token and try again."
    exit 1
fi

echo ""
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}${BOLD}  Hookbuster is running!${NC}"
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  Now trigger Webex events to see them forwarded:"
echo ""
echo "    • Send a message in any Webex space"
echo "    • Create a new space"
echo "    • Add/remove someone from a space"
echo "    • Submit an adaptive card"
echo ""
echo "  Events will appear below as JSON."
echo "  Press Ctrl+C to stop."
echo ""

# ── Wait for hookbuster to exit ──────────────────────────────────────────
wait "$HOOKBUSTER_PID" 2>/dev/null || true
