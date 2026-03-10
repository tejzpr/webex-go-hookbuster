#!/bin/bash
# ──────────────────────────────────────────────────────────────────────────
# Hookbuster Test Script
#
# Two test modes:
#   firehose   – single receiver, env-var deployment mode (all resources)
#   roundrobin – three receivers, config-file mode with round-robin LB
#
# Usage:
#   ./test.sh                                  # prompts for mode & token
#   ./test.sh firehose                         # firehose mode, prompts for token
#   ./test.sh roundrobin                       # roundrobin mode, prompts for token
#   WEBEX_TOKEN=<token> ./test.sh firehose     # fully automated
#   WEBEX_TOKEN=<token> ./test.sh roundrobin   # fully automated
# ──────────────────────────────────────────────────────────────────────────

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="$SCRIPT_DIR/hookbuster"
HOOKBUSTER_PID=""
RECV_PIDS=()
CONFIG_FILE=""

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

    for pid in "${RECV_PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null
            wait "$pid" 2>/dev/null || true
        fi
    done
    if [ ${#RECV_PIDS[@]} -gt 0 ]; then
        ok "Receiver(s) stopped"
    fi

    if [ -n "$CONFIG_FILE" ] && [ -f "$CONFIG_FILE" ]; then
        rm -f "$CONFIG_FILE"
    fi

    ok "Done."
}
trap cleanup EXIT INT TERM

# ── Start a Python HTTP receiver on a given port ─────────────────────────
start_receiver() {
    local port=$1
    local label=$2
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
            print(f'===== [$label:$port] RECEIVED: {resource}:{event} =====')
            print(json.dumps(data, indent=2))
            print(f'=================================================')
            print(f'')
            sys.stdout.flush()
        except Exception as e:
            print(f'[$label:$port] parse error: {e}')
            sys.stdout.flush()
        self.send_response(200)
        self.end_headers()

    def do_HEAD(self):
        self.send_response(200)
        self.end_headers()

    def log_message(self, format, *args):
        pass  # suppress default access logs

print(f'[$label] Listening on port $port ...')
sys.stdout.flush()
HTTPServer(('', $port), Handler).serve_forever()
" &
    RECV_PIDS+=($!)
}

# ── Step 1: Build ────────────────────────────────────────────────────────
info "Building hookbuster..."
cd "$SCRIPT_DIR"
go build -o "$BINARY" .
ok "Built: $BINARY"

# ── Step 2: Select test mode ─────────────────────────────────────────────
MODE="${1:-}"
if [ -z "$MODE" ]; then
    echo ""
    echo -e "${BOLD}Select test mode:${NC}"
    echo "  1) firehose   – single receiver, env-var deployment mode"
    echo "  2) roundrobin – three receivers, config-file round-robin LB"
    echo -n "> "
    read -r MODE_CHOICE
    case "$MODE_CHOICE" in
        1|firehose)   MODE="firehose" ;;
        2|roundrobin) MODE="roundrobin" ;;
        *)            err "Invalid choice. Use 'firehose' or 'roundrobin'."; exit 1 ;;
    esac
fi

if [ "$MODE" != "firehose" ] && [ "$MODE" != "roundrobin" ]; then
    err "Unknown mode '$MODE'. Use 'firehose' or 'roundrobin'."
    exit 1
fi

info "Test mode: $MODE"

# ── Step 3: Get Webex token ──────────────────────────────────────────────
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

# ── Step 4: Start receiver(s) and hookbuster ─────────────────────────────

if [ "$MODE" = "firehose" ]; then
    # ── Firehose: single receiver on port 8080 ──────────────────────────
    RECV_PORT=8080
    info "Starting receiver on port $RECV_PORT..."
    start_receiver "$RECV_PORT" "receiver"
    sleep 1

    if ! kill -0 "${RECV_PIDS[0]}" 2>/dev/null; then
        err "Failed to start receiver. Is port $RECV_PORT already in use?"
        exit 1
    fi
    ok "Receiver running (PID ${RECV_PIDS[0]})"

    info "Starting hookbuster (firehose mode → all resources, all events)..."
    echo ""

    TOKEN="$TOKEN" PORT="$RECV_PORT" TARGET="localhost" "$BINARY" &
    HOOKBUSTER_PID=$!

else
    # ── Roundrobin: three receivers on ports 5001–5003 ──────────────────
    RR_PORTS=(5001 5002 5003)

    for port in "${RR_PORTS[@]}"; do
        info "Starting receiver on port $port..."
        start_receiver "$port" "rr"
    done
    sleep 1

    for i in "${!RECV_PIDS[@]}"; do
        if ! kill -0 "${RECV_PIDS[$i]}" 2>/dev/null; then
            err "Failed to start receiver on port ${RR_PORTS[$i]}."
            exit 1
        fi
        ok "Receiver running on port ${RR_PORTS[$i]} (PID ${RECV_PIDS[$i]})"
    done

    # Generate a temporary config file
    CONFIG_FILE="$(mktemp "${TMPDIR:-/tmp}/hookbuster-test-XXXXXX.yml")"
    cat > "$CONFIG_FILE" <<EOF
pipelines:
  - name: "roundrobin-test"
    token_env: "HOOKBUSTER_TEST_TOKEN"
    mode: "roundrobin"
    events: "all"
    targets:
      - url: "http://localhost:5001"
      - url: "http://localhost:5002"
      - url: "http://localhost:5003"
EOF
    ok "Generated config: $CONFIG_FILE"

    info "Starting hookbuster (roundrobin mode → 3 targets)..."
    echo ""

    HOOKBUSTER_TEST_TOKEN="$TOKEN" "$BINARY" -c "$CONFIG_FILE" &
    HOOKBUSTER_PID=$!
fi

sleep 3

if ! kill -0 "$HOOKBUSTER_PID" 2>/dev/null; then
    err "Hookbuster exited unexpectedly. Check the token and try again."
    exit 1
fi

echo ""
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}${BOLD}  Hookbuster is running!  (mode: $MODE)${NC}"
echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  Now trigger Webex events to see them forwarded:"
echo ""
echo "    • Send a message in any Webex space"
echo "    • Create a new space"
echo "    • Add/remove someone from a space"
echo "    • Submit an adaptive card"
echo ""
if [ "$MODE" = "roundrobin" ]; then
echo "  Events will be load-balanced across ports 5001, 5002, 5003."
else
echo "  Events will appear on port 8080."
fi
echo "  Press Ctrl+C to stop."
echo ""

# ── Wait for hookbuster to exit ──────────────────────────────────────────
wait "$HOOKBUSTER_PID" 2>/dev/null || true
