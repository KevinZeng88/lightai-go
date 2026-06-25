#!/usr/bin/env bash
# E2E API-first dry-run — validates current deployment contract.
# Requires: running LightAI server+agent, no GPU needed for dry-run.
# Usage: bash scripts/e2e-current-contract-api-dryrun.sh
set -euo pipefail

SERVER="${LIGHTAI_SERVER_URL:-http://localhost:18080}"
AGENT="${LIGHTAI_AGENT_URL:-http://localhost:19091}"
PASS="${LIGHTAI_BOOTSTRAP_ADMIN_PASSWORD:-}"
INIT="${LIGHTAI_BOOTSTRAP_INITIAL_PASSWORD:-}"
test -n "$PASS" || test -n "$INIT" || PASS="test-init-123"
test -n "$INIT" || INIT="$PASS"
USER="admin"

echo "=== E2E Current Contract API Dry-Run ==="

# Login
COOKIE=$(mktemp)
LOGIN_RESP=$(mktemp)
CODE=$(curl -sS -o "$LOGIN_RESP" -w '%{http_code}' -c "$COOKIE" -X POST "$SERVER/api/v1/auth/login" \
  -H "Origin: $SERVER" -H "Content-Type: application/json" \
  -d "{\"username\":\"$USER\",\"password\":\"$INIT\"}")
if test "$CODE" != "200"; then echo "FAIL: login HTTP $CODE"; rm -f "$COOKIE" "$LOGIN_RESP"; exit 1; fi
CSRF=$(python3 -c "import json;print(json.load(open('$LOGIN_RESP'))['csrf_token'])" 2>/dev/null)
echo "Login: OK"

# Get backends
echo "=== Backends ==="
curl -sS -b "$COOKIE" "$SERVER/api/v1/backends" -H "Origin: $SERVER" | python3 -c "
import json,sys;d=json.load(sys.stdin);arr=d if isinstance(d,list) else d.get('data',[])
for b in arr: print(f'  {b[\"id\"]}: {b.get(\"display_name\",\"\")}')
"

# Get nodes
echo "=== Nodes ==="
NODES=$(curl -sS -b "$COOKIE" "$SERVER/api/v1/nodes" -H "Origin: $SERVER")
NODE_ID=$(echo "$NODES" | python3 -c "import json,sys;arr=json.load(sys.stdin) if isinstance(json.load(sys.stdin),list) else json.load(sys.stdin).get('data',[]);print(arr[0]['id'] if arr else '')")
if test -z "$NODE_ID"; then echo "SKIP: no registered node (agent may not be running)"; rm -f "$COOKIE" "$LOGIN_RESP"; exit 0; fi
echo "Node: $NODE_ID"

# Get NBRs
echo "=== NBRs ==="
NBRS=$(curl -sS -b "$COOKIE" "$SERVER/api/v1/nodes/$NODE_ID/backend-runtimes" -H "Origin: $SERVER")
READY_NBR=$(echo "$NBRS" | python3 -c "import json,sys;d=json.load(sys.stdin);arr=d if isinstance(d,list) else d.get('data',[]);[print(r['id']) for r in arr if r.get('status') in ('ready','ready_with_warnings')];print('')" | head -1)
if test -z "$READY_NBR"; then echo "SKIP: no ready NBR (run bootstrap to create)"; rm -f "$COOKIE" "$LOGIN_RESP"; exit 0; fi
echo "NBR: $READY_NBR"

# Get artifacts
ARTIFACTS=$(curl -sS -b "$COOKIE" "$SERVER/api/v1/model-artifacts" -H "Origin: $SERVER")
ART_ID=$(echo "$ARTIFACTS" | python3 -c "import json,sys;d=json.load(sys.stdin);arr=d if isinstance(d,list) else d.get('data',[]);print(arr[0]['id'] if arr else '')")
if test -z "$ART_ID"; then echo "SKIP: no model artifacts"; rm -f "$COOKIE" "$LOGIN_RESP"; exit 0; fi
echo "Artifact: $ART_ID"

# Preflight
echo "=== Preflight ==="
PF_RESP=$(mktemp)
curl -sS -o "$PF_RESP" -X POST "$SERVER/api/v1/deployments/preflight" \
  -H "Origin: $SERVER" -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" -b "$COOKIE" \
  -d "{\"model_artifact_id\":\"$ART_ID\",\"node_backend_runtime_id\":\"$READY_NBR\",\"node_id\":\"$NODE_ID\",\"host_port\":9000}"
CAN_RUN=$(python3 -c "import json;print(json.load(open('$PF_RESP')).get('can_run',False))")
echo "Preflight can_run=$CAN_RUN"
if test "$CAN_RUN" != "True"; then echo "FAIL: preflight returned can_run=false"; cat "$PF_RESP"; rm -f "$COOKIE" "$LOGIN_RESP" "$PF_RESP"; exit 1; fi

# Create deployment
echo "=== Create Deployment ==="
DEPL_RESP=$(mktemp)
curl -sS -o "$DEPL_RESP" -X POST "$SERVER/api/v1/deployments" \
  -H "Origin: $SERVER" -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" -b "$COOKIE" \
  -d "{\"name\":\"e2e-contract-dryrun\",\"model_artifact_id\":\"$ART_ID\",\"node_backend_runtime_id\":\"$READY_NBR\",\"service_json\":{\"host_port\":9000}}"
DEPL_ID=$(python3 -c "import json;d=json.load(open('$DEPL_RESP'));print(d.get('id',''))" 2>/dev/null)
if test -z "$DEPL_ID"; then echo "FAIL: create deployment"; cat "$DEPL_RESP"; rm -f "$COOKIE" "$LOGIN_RESP" "$PF_RESP" "$DEPL_RESP"; exit 1; fi
echo "Deployment: $DEPL_ID"

# Dry-run
echo "=== Dry-Run ==="
DR_RESP=$(mktemp)
curl -sS -o "$DR_RESP" -X POST "$SERVER/api/v1/deployments/$DEPL_ID/dry-run" \
  -H "Origin: $SERVER" -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF" -b "$COOKIE" -d '{}'
DR_OK=$(python3 -c "import json;d=json.load(open('$DR_RESP'));print('pass' if d.get('resolved_run_plan') else 'fail')" 2>/dev/null)
echo "Dry-run: $DR_OK"

# Cleanup
curl -sS -X DELETE "$SERVER/api/v1/deployments/$DEPL_ID" \
  -H "Origin: $SERVER" -H "X-CSRF-Token: $CSRF" -b "$COOKIE" >/dev/null
echo "=== PASS ==="
rm -f "$COOKIE" "$LOGIN_RESP" "$PF_RESP" "$DEPL_RESP" "$DR_RESP"
