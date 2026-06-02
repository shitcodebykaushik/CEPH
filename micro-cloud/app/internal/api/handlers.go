package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shiiit/micro-cloud/internal/database"
	"github.com/shiiit/micro-cloud/internal/orchestrator"
)

type Handler struct {
	Engine *orchestrator.Engine
}

func NewHandler(eng *orchestrator.Engine) *Handler {
	return &Handler{Engine: eng}
}

func jsonResp(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, status int, msg string) {
	jsonResp(w, status, ErrorResponse{Error: msg})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		jsonErr(w, http.StatusBadRequest, "name is required")
		return
	}

	id := uuid.New().String()
	quota := req.StorageQuotaBytes
	if quota <= 0 {
		quota = 10 * 1024 * 1024 * 1024
	}

	_, err := database.Exec(
		`INSERT INTO tenants (id, name, storage_quota_bytes) VALUES (?, ?, ?)`,
		id, req.Name, quota,
	)
	if err != nil {
		log.Printf("[API] CreateTenant error: %v", err)
		jsonErr(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}

	jsonResp(w, http.StatusCreated, CreateTenantResponse{
		ID:                id,
		Name:              req.Name,
		StorageQuotaBytes: quota,
	})
}

func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	rows, err := database.Query(`SELECT id, name, active_instances, storage_quota_bytes, created_at FROM tenants`)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var tenants []TenantResponse
	for rows.Next() {
		var t TenantResponse
		if err := rows.Scan(&t.ID, &t.Name, &t.ActiveInstances, &t.StorageQuota, &t.CreatedAt); err != nil {
			continue
		}
		tenants = append(tenants, t)
	}
	if tenants == nil {
		tenants = []TenantResponse{}
	}
	jsonResp(w, http.StatusOK, tenants)
}

func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row := database.QueryRow(`SELECT id, name, active_instances, storage_quota_bytes, created_at FROM tenants WHERE id = ?`, id)

	var t TenantResponse
	if err := row.Scan(&t.ID, &t.Name, &t.ActiveInstances, &t.StorageQuota, &t.CreatedAt); err != nil {
		jsonErr(w, http.StatusNotFound, "tenant not found")
		return
	}
	jsonResp(w, http.StatusOK, t)
}

func (h *Handler) CreateVolume(w http.ResponseWriter, r *http.Request) {
	var req CreateVolumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TenantID == "" || req.SizeMB <= 0 {
		jsonErr(w, http.StatusBadRequest, "tenant_id and size_mb are required")
		return
	}

	volumeID := uuid.New().String()

	_, err := database.Exec(
		`INSERT INTO volumes (id, tenant_id, size_mb, status) VALUES (?, ?, ?, 'creating')`,
		volumeID, req.TenantID, req.SizeMB,
	)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to create volume record")
		return
	}

	go func() {
		if err := h.Engine.CreateVolume(req.TenantID, volumeID, req.SizeMB); err != nil {
			log.Printf("[VOLUME] Provisioning failed for %s: %v", volumeID, err)
			database.Exec(`UPDATE volumes SET status = 'failed' WHERE id = ?`, volumeID)
		}
	}()

	jsonResp(w, http.StatusAccepted, map[string]string{
		"volume_id": volumeID,
		"status":    "creating",
	})
}

func (h *Handler) ListVolumes(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	var rows_query string
	var args []any
	if tenantID != "" {
		rows_query = `SELECT id, tenant_id, size_mb, status, COALESCE(mount_path,''), created_at FROM volumes WHERE tenant_id = ?`
		args = append(args, tenantID)
	} else {
		rows_query = `SELECT id, tenant_id, size_mb, status, COALESCE(mount_path,''), created_at FROM volumes`
	}

	rows, err := database.Query(rows_query, args...)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var volumes []VolumeResponse
	for rows.Next() {
		var v VolumeResponse
		if err := rows.Scan(&v.ID, &v.TenantID, &v.SizeMB, &v.Status, &v.MountPath, &v.CreatedAt); err != nil {
			continue
		}
		volumes = append(volumes, v)
	}
	if volumes == nil {
		volumes = []VolumeResponse{}
	}
	jsonResp(w, http.StatusOK, volumes)
}

func (h *Handler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TenantID == "" {
		jsonErr(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	if req.Image == "" {
		req.Image = "ubuntu:22.04"
	}
	if req.SizeMB <= 0 {
		req.SizeMB = 5120
	}

	workspaceID := uuid.New().String()
	volumeID := uuid.New().String()

	_, err := database.Exec(
		`INSERT INTO workspaces (id, tenant_id, image, associated_volume_id, status, requester_name, requester_email) VALUES (?, ?, ?, ?, 'launching', ?, ?)`,
		workspaceID, req.TenantID, req.Image, volumeID, req.RequesterName, req.RequesterEmail,
	)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to create workspace record")
		return
	}

	_, err = database.Exec(
		`INSERT INTO volumes (id, tenant_id, size_mb, status) VALUES (?, ?, ?, 'creating')`,
		volumeID, req.TenantID, req.SizeMB,
	)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to create volume record")
		return
	}

	go func() {
		if err := h.Engine.LaunchWorkspace(workspaceID, volumeID, req.TenantID, req.SizeMB, req.Image); err != nil {
			log.Printf("[WORKSPACE] Provisioning failed for %s: %v", workspaceID, err)
			database.Exec(`UPDATE workspaces SET status = 'failed' WHERE id = ?`, workspaceID)
		}
	}()

	jsonResp(w, http.StatusAccepted, map[string]string{
		"workspace_id": workspaceID,
		"volume_id":    volumeID,
		"status":       "launching",
	})
}

func (h *Handler) RequestWorkspace(w http.ResponseWriter, r *http.Request) {
	var req RequestWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Email == "" {
		jsonErr(w, http.StatusBadRequest, "name and email are required")
		return
	}
	if req.Image == "" {
		req.Image = "alpine:latest"
	}
	if req.SizeMB <= 0 {
		req.SizeMB = 512
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		row := database.QueryRow(`SELECT id FROM tenants ORDER BY created_at ASC LIMIT 1`)
		if err := row.Scan(&tenantID); err != nil {
			jsonErr(w, http.StatusNotFound, "no tenant available; create one first")
			return
		}
	}

	workspaceID := uuid.New().String()
	volumeID := uuid.New().String()

	_, err := database.Exec(
		`INSERT INTO workspaces (id, tenant_id, image, associated_volume_id, status, requester_name, requester_email) VALUES (?, ?, ?, ?, 'launching', ?, ?)`,
		workspaceID, tenantID, req.Image, volumeID, req.Name, req.Email,
	)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}

	_, err = database.Exec(
		`INSERT INTO volumes (id, tenant_id, size_mb, status) VALUES (?, ?, ?, 'creating')`,
		volumeID, tenantID, req.SizeMB,
	)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "failed to create volume")
		return
	}

	go func() {
		if err := h.Engine.LaunchWorkspace(workspaceID, volumeID, tenantID, req.SizeMB, req.Image); err != nil {
			log.Printf("[WORKSPACE] Provisioning failed for %s: %v", workspaceID, err)
			database.Exec(`UPDATE workspaces SET status = 'failed' WHERE id = ?`, workspaceID)
		}
	}()

	log.Printf("[WORKSPACE] User %s (%s) requested workspace %s", req.Name, req.Email, workspaceID)
	jsonResp(w, http.StatusAccepted, map[string]string{
		"workspace_id": workspaceID,
		"volume_id":    volumeID,
		"status":       "launching",
	})
}

func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	var rows_query string
	var args []any
	if tenantID != "" {
		rows_query = `SELECT id, tenant_id, COALESCE(container_id,''), image, COALESCE(associated_volume_id,''), COALESCE(internal_ip,''), COALESCE(port,0), status, COALESCE(requester_name,''), COALESCE(requester_email,''), created_at FROM workspaces WHERE tenant_id = ? ORDER BY created_at DESC`
		args = append(args, tenantID)
	} else {
		rows_query = `SELECT id, tenant_id, COALESCE(container_id,''), image, COALESCE(associated_volume_id,''), COALESCE(internal_ip,''), COALESCE(port,0), status, COALESCE(requester_name,''), COALESCE(requester_email,''), created_at FROM workspaces ORDER BY created_at DESC`
	}

	rows, err := database.Query(rows_query, args...)
	if err != nil {
		jsonErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	var workspaces []WorkspaceResponse
	for rows.Next() {
		var ws WorkspaceResponse
		if err := rows.Scan(&ws.ID, &ws.TenantID, &ws.ContainerID, &ws.Image, &ws.VolumeID, &ws.InternalIP, &ws.Port, &ws.Status, &ws.RequesterName, &ws.RequesterEmail, &ws.CreatedAt); err != nil {
			continue
		}
		workspaces = append(workspaces, ws)
	}
	if workspaces == nil {
		workspaces = []WorkspaceResponse{}
	}
	jsonResp(w, http.StatusOK, workspaces)
}

func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	row := database.QueryRow(
		`SELECT id, tenant_id, COALESCE(container_id,''), image, COALESCE(associated_volume_id,''), COALESCE(internal_ip,''), COALESCE(port,0), status, COALESCE(requester_name,''), COALESCE(requester_email,''), created_at FROM workspaces WHERE id = ?`, id,
	)

	var ws WorkspaceResponse
	if err := row.Scan(&ws.ID, &ws.TenantID, &ws.ContainerID, &ws.Image, &ws.VolumeID, &ws.InternalIP, &ws.Port, &ws.Status, &ws.RequesterName, &ws.RequesterEmail, &ws.CreatedAt); err != nil {
		jsonErr(w, http.StatusNotFound, "workspace not found")
		return
	}

	ws.SSHAddress = fmt.Sprintf("localhost:%d", ws.Port)
	jsonResp(w, http.StatusOK, ws)
}

func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	row := database.QueryRow(`SELECT container_id, associated_volume_id FROM workspaces WHERE id = ?`, id)
	var containerID, volumeID string
	if err := row.Scan(&containerID, &volumeID); err != nil {
		jsonErr(w, http.StatusNotFound, "workspace not found")
		return
	}

	database.Exec(`UPDATE workspaces SET status = 'stopping' WHERE id = ?`, id)

	go func() {
		if err := h.Engine.TerminateWorkspace(id, containerID, volumeID); err != nil {
			log.Printf("[WORKSPACE] Termination failed for %s: %v", id, err)
		}
	}()

	jsonResp(w, http.StatusAccepted, map[string]string{
		"workspace_id": id,
		"status":       "stopping",
	})
}


