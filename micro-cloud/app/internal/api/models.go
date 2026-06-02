package api

import "time"

type CreateTenantRequest struct {
	Name              string `json:"name"`
	StorageQuotaBytes int64  `json:"storage_quota_bytes,omitempty"`
}

type CreateTenantResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	StorageQuotaBytes int64  `json:"storage_quota_bytes"`
}

type TenantResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ActiveInstances int    `json:"active_instances"`
	StorageQuota    int64  `json:"storage_quota_bytes"`
	CreatedAt       string `json:"created_at"`
}

type VolumeResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	SizeMB    int    `json:"size_mb"`
	Status    string `json:"status"`
	MountPath string `json:"mount_path,omitempty"`
	CreatedAt string `json:"created_at"`
}

type CreateVolumeRequest struct {
	TenantID string `json:"tenant_id"`
	SizeMB   int    `json:"size_mb"`
}

type CreateWorkspaceRequest struct {
	TenantID       string `json:"tenant_id"`
	Image          string `json:"image"`
	SizeMB         int    `json:"size_mb"`
	RequesterName  string `json:"requester_name,omitempty"`
	RequesterEmail string `json:"requester_email,omitempty"`
}

type RequestWorkspaceRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Image string `json:"image"`
	SizeMB int   `json:"size_mb"`
}

type WorkspaceResponse struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	ContainerID    string `json:"container_id,omitempty"`
	Image          string `json:"image"`
	VolumeID       string `json:"volume_id,omitempty"`
	InternalIP     string `json:"internal_ip,omitempty"`
	Port           int    `json:"port,omitempty"`
	Status         string `json:"status"`
	SSHAddress     string `json:"ssh_address,omitempty"`
	RequesterName  string `json:"requester_name,omitempty"`
	RequesterEmail string `json:"requester_email,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func nowStr() string {
	return time.Now().UTC().Format(time.RFC3339)
}
