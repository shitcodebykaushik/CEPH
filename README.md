# The motive 
- By building this project what we are writing is the software architecture in the GO,seperating the API Brain,the compute Node and the storage pool it is the exact design paradigm used by the entrprise cloud provider And with the container technology our host can share multiple friends  sandboxes .

# Micro-Cloud Provider Engine

A lightweight multi-tenant cloud platform that runs on a single machine using **Ceph** for block storage and **Docker** for compute workspaces.

Each tenant gets isolated workspaces (containers) with persistent volumes backed by Ceph RBD.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Docker Host                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │  ceph-mon │  │  ceph-osd │  │orchestrator-api  │   │
│  │           │  │           │  │  ┌────────────┐  │   │
│  │           │  │           │  │  │  Web UI    │  │   │
│  │           │  │           │  │  │  :8080     │  │   │
│  │           │  │           │  │  └────────────┘  │   │
│  └──────────┘  └──────────┘  └──────────────────┘   │
│                                        │             │
│  ┌─────────────────────────────────────┘             │
│  │                                                    │
│  ┌▼────────────────────────────────────────────────┐  │
│  │         Tenant Workspaces (Docker containers)    │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐      │  │
│  │  │ workspace1 │  │ workspace2 │  │ workspace3 │  │  │
│  │  │  :10001   │  │  :10002   │  │  :10003   │  │  │
│  │  │  /workspace│  │  /workspace│  │  /workspace│  │  │
│  │  └──────────┘  └──────────┘  └──────────┘      │  │
│  └────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

## Prerequisites

- Docker Engine 24+ with Docker Compose plugin
- ~10 GB free disk space
- Linux (recommended) or macOS

## Quick Start

```bash
# 1. Start everything
docker compose up -d

# 2. Check health
curl http://localhost:8080/health

# 3. Open the web UI
#    Browser → http://localhost:8080
```

The web UI lets you manage tenants, volumes, and workspaces visually.

## API Usage

### Tenants

```bash
# Create a tenant
curl -X POST http://localhost:8080/api/v1/tenants \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "storage_quota_bytes": 10737418240}'

# List tenants
curl http://localhost:8080/api/v1/tenants
```

### Volumes

```bash
# Create a volume (async — returns 202)
curl -X POST http://localhost:8080/api/v1/volumes \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id": "<tenant_id>", "size_mb": 1024}'

# List volumes
curl http://localhost:8080/api/v1/volumes
```

### Workspaces

```bash
# Launch a workspace (async — returns 202)
curl -X POST http://localhost:8080/api/v1/workspaces \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id": "<tenant_id>", "image": "alpine:latest", "size_mb": 512}'

# Check status
curl http://localhost:8080/api/v1/workspaces/<workspace_id>

# List all workspaces
curl http://localhost:8080/api/v1/workspaces

# Terminate a workspace
curl -X DELETE http://localhost:8080/api/v1/workspaces/<workspace_id>
```

## Using a Workspace

Once a workspace is `running`, access it with Docker exec:

```bash
# 1. Find the container ID
docker ps | grep orchestrator-api

# Or from the workspace details in web UI or API

# 2. Open a shell inside the workspace
docker exec -it <container_id> sh

# 3. Your persistent volume is at /workspace
cd /workspace

# 4. Install tools (Alpine uses apk)
apk add python3 git curl vim

# 5. Do your work — files in /workspace persist
git clone <your-repo>
python3 script.py
```

**Important:** Files outside `/workspace` are temporary and lost if the container restarts. `/workspace` is backed by Ceph and persists.

### With SSH

The default `alpine:latest` image has no SSH server. To use SSH, launch the workspace with an SSH-enabled image like:

```bash
curl -X POST http://localhost:8080/api/v1/workspaces \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id": "<tenant_id>", "image": "rastasheep/ubuntu-sshd:18.04", "size_mb": 512}'
```

Then SSH in:

```bash
ssh root@localhost -p <port>
# Password: root
```

## Example Workflow

```bash
# 1. Create a tenant
TENANT_ID=$(curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H 'Content-Type: application/json' \
  -d '{"name": "Alice", "storage_quota_bytes": 10737418240}' | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

# 2. Launch a workspace
WS_RESP=$(curl -s -X POST http://localhost:8080/api/v1/workspaces \
  -H 'Content-Type: application/json' \
  -d "{\"tenant_id\": \"$TENANT_ID\", \"image\": \"alpine:latest\", \"size_mb\": 512}")
WORKSPACE_ID=$(echo "$WS_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['workspace_id'])")

# 3. Wait for it to be running
sleep 15
curl http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID

# 4. Exec into it
CONTAINER_ID=$(curl -s http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID | python3 -c "import sys,json; print(json.load(sys.stdin)['container_id'][:12])")
docker exec -it $CONTAINER_ID sh

# 5. Terminate when done
curl -X DELETE http://localhost:8080/api/v1/workspaces/$WORKSPACE_ID
```

## Tear Down

```bash
# Stop everything and remove containers
docker compose down

# Remove Ceph data (optional)
sudo rm -rf ceph_config ceph_bootstrap osd_storage

# Remove volume data (optional)
sudo rm -rf /tmp/micro-cloud/volumes
```
