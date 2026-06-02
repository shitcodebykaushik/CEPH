CREATE TABLE IF NOT EXISTS tenants (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    active_instances INTEGER DEFAULT 0,
    storage_quota_bytes INTEGER DEFAULT 10737418240, -- 10 GB default
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS volumes (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT NOT NULL,
    size_mb         INTEGER NOT NULL,
    status          TEXT DEFAULT 'creating', -- creating, available, attached, deleting
    pool_name       TEXT DEFAULT 'rbd',
    image_name      TEXT NOT NULL,
    device_path     TEXT,
    mount_path      TEXT,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS workspaces (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT NOT NULL,
    container_id    TEXT,
    image          TEXT NOT NULL DEFAULT 'ubuntu:22.04',
    associated_volume_id TEXT,
    internal_ip     TEXT,
    port            INTEGER,
    status          TEXT DEFAULT 'launching', -- launching, running, stopping, stopped, failed
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (associated_volume_id) REFERENCES volumes(id) ON DELETE SET NULL
);
