#!/bin/sh
# LightAI Go - Status Check
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RLS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$RLS_ROOT"

echo "=== LightAI Go 状态 ==="
echo ""

check_proc() {
  local name="$1" pidfile="$2" port="$3" health_url="$4"
  if [ -f "$pidfile" ]; then
    PID=$(cat "$pidfile")
    if kill -0 "$PID" 2>/dev/null; then
      printf "  %-13s 运行中 (PID %s, 端口 %s)\n" "$name" "$PID" "$port"
      if [ -n "$health_url" ]; then
        if curl -sf --max-time 2 "$health_url" >/dev/null 2>&1; then
          printf "    Health: OK\n"
        else
          printf "    Health: 无响应\n"
        fi
      fi
    else
      printf "  %-13s 未运行 (已清理残留 PID)\n" "$name"
      rm -f "$pidfile"
    fi
  else
    printf "  %-13s 未运行\n" "$name"
  fi
}

check_proc "Server"      run/server.pid      "8080" "http://127.0.0.1:8080/healthz"
check_proc "Agent"       run/agent.pid       "9091" "http://127.0.0.1:9091/healthz"
check_proc "Prometheus"  run/prometheus.pid  "9090" "http://127.0.0.1:9090/-/ready"
check_proc "Grafana"     run/grafana.pid     "3000" "http://127.0.0.1:3000/api/health"

echo ""
echo "--- 访问地址 ---"
echo "  LightAI Web: http://127.0.0.1:8080/"
echo "  Prometheus:  http://127.0.0.1:9090/"
echo "  Grafana:     http://127.0.0.1:3000/"
echo ""
echo "  局域网访问请将 127.0.0.1 替换为服务器 IP。"

echo ""
echo "--- 最近日志 ---"
for f in logs/server-stdout.log logs/agent-stdout.log logs/prometheus.log logs/grafana.log; do
  if [ -f "$f" ]; then
    echo "  [$f]"
    tail -2 "$f" | sed 's/^/    /'
  fi
done
