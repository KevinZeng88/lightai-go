#!/bin/sh
# LightAI Go - Status Check
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RELEASE_ROOT"

echo "=== LightAI Go Status ==="
echo "Root: $RELEASE_ROOT"
echo ""

check_proc() {
  local name="$1" pidfile="$2" port="$3" health_url="$4"
  if [ -f "$pidfile" ]; then
    PID=$(cat "$pidfile")
    if kill -0 "$PID" 2>/dev/null; then
      printf "  %-8s RUNNING  (PID %s, port %s)\n" "$name" "$PID" "$port"
      if [ -n "$health_url" ]; then
        if curl -sf "$health_url" >/dev/null 2>&1; then
          printf "    Health: OK\n"
        else
          printf "    Health: NOT RESPONDING\n"
        fi
      fi
      return 0
    else
      printf "  %-8s STOPPED  (stale PID)\n" "$name"
      return 1
    fi
  else
    printf "  %-8s STOPPED\n" "$name"
    return 1
  fi
}

check_proc "Server" run/server.pid "8080" "http://127.0.0.1:8080/healthz"
echo ""
check_proc "Agent"  run/agent.pid  "19091" "http://127.0.0.1:19091/healthz"

echo ""
echo "--- Recent Logs (server) ---"
if [ -f logs/server-stdout.log ]; then
  tail -5 logs/server-stdout.log
fi
echo ""
echo "--- Recent Logs (agent) ---"
if [ -f logs/agent-stdout.log ]; then
  tail -5 logs/agent-stdout.log
fi
