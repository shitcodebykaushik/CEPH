#!/bin/bash
set -e

echo "=== Micro-Cloud Setup ==="

# Start infrastructure
echo "[1/5] Starting Ceph cluster + orchestrator..."
docker compose up -d
sleep 10

# Wait for orchestrator
echo "[2/5] Waiting for orchestrator API..."
until curl -s http://localhost:8080/health > /dev/null 2>&1; do sleep 2; done
echo "      API ready."

# Create tenant
echo "[3/5] Creating tenant..."
TENANT_ID=$(curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H 'Content-Type: application/json' \
  -d '{"name":"default","storage_quota_bytes":10737418240}' | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "      Tenant ID: $TENANT_ID"

# Launch workspace
echo "[4/5] Launching workspace..."
WS_RESP=$(curl -s -X POST http://localhost:8080/api/v1/workspaces \
  -H 'Content-Type: application/json' \
  -d "{\"tenant_id\":\"$TENANT_ID\",\"image\":\"alpine:latest\",\"size_mb\":256}")
WORKSPACE_ID=$(echo "$WS_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['workspace_id'])")
echo "      Workspace ID: $WORKSPACE_ID"

# Wait for it to be running
echo "[5/5] Waiting for workspace to be ready..."
for i in $(seq 1 30); do
  STATUS=$(curl -s http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])")
  if [ "$STATUS" = "running" ]; then
    PORT=$(curl -s http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID | python3 -c "import sys,json; print(json.load(sys.stdin)['port'])")
    CONTAINER_ID=$(curl -s http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID | python3 -c "import sys,json; print(json.load(sys.stdin)['container_id'][:12])")
    echo ""
    echo "=== Done ==="
    echo "Web UI:    http://localhost:8080"
    echo "Workspace: docker exec -it $CONTAINER_ID sh"
    echo "Volume at: /workspace"
    exit 0
  fi
  sleep 2
done

echo "ERROR: Workspace did not become running. Check docker logs orchestrator-api"
exit 1
