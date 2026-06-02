package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/client"
	"github.com/shiiit/micro-cloud/internal/database"
	"github.com/shiiit/micro-cloud/internal/storage"
)

type Engine struct {
	Minio  *storage.MinioManager
	Docker *client.Client
}

func NewEngine(mm *storage.MinioManager, dc *client.Client) *Engine {
	return &Engine{
		Minio:  mm,
		Docker: dc,
	}
}

func (e *Engine) CreateVolume(tenantID, volumeID string, sizeMB int) error {
	if err := storage.CreateVolumeDir(volumeID); err != nil {
		return fmt.Errorf("create volume dir: %w", err)
	}

	if _, err := database.Exec(
		`UPDATE volumes SET status = 'available' WHERE id = ?`,
		volumeID,
	); err != nil {
		return fmt.Errorf("update volume status: %w", err)
	}

	log.Printf("[ENGINE] Volume %s ready (%d MB)", volumeID, sizeMB)
	return nil
}

func (e *Engine) LaunchWorkspace(workspaceID, volumeID, tenantID string, sizeMB int, containerImage string) error {
	log.Printf("[ENGINE] Launching workspace %s for tenant %s", workspaceID, tenantID)

	mountPath := storage.VolumeDirPath(volumeID)

	if err := storage.CreateVolumeDir(volumeID); err != nil {
		database.Exec(`UPDATE volumes SET status = 'failed' WHERE id = ?`, volumeID)
		return fmt.Errorf("create volume dir: %w", err)
	}

	database.Exec(
		`UPDATE volumes SET status = 'attached', mount_path = ? WHERE id = ?`,
		mountPath, volumeID,
	)

	log.Printf("[ENGINE] Pulling image %s ...", containerImage)
	if err := storage.PullImage(context.Background(), e.Docker, containerImage); err != nil {
		log.Printf("[ENGINE] Pull error (may already exist): %v", err)
	}

	containerResult, err := storage.LaunchContainer(context.Background(), e.Docker, storage.ContainerConfig{
		Image:      containerImage,
		VolumeBind: mountPath,
		EntryPoint: []string{"sleep", "infinity"},
	})
	if err != nil {
		log.Printf("[ENGINE] Container launch failed: %v", err)
		e.cleanupVolume(volumeID)
		return fmt.Errorf("launch container: %w", err)
	}

	database.Exec(
		`UPDATE workspaces SET container_id = ?, port = ?, status = 'running' WHERE id = ?`,
		containerResult.ContainerID, containerResult.Port, workspaceID,
	)

	database.Exec(
		`UPDATE tenants SET active_instances = active_instances + 1 WHERE id = ?`,
		tenantID,
	)

	hostname, _ := os.Hostname()
	log.Printf("[ENGINE] Workspace %s is RUNNING (container: %s, ssh: %s:%d)",
		workspaceID, containerResult.ContainerID[:12], hostname, containerResult.Port)

	return nil
}

func (e *Engine) TerminateWorkspace(workspaceID, containerID, volumeID string) error {
	ctx := context.Background()

	if containerID != "" {
		if err := storage.StopContainer(ctx, e.Docker, containerID); err != nil {
			log.Printf("[ENGINE] Stop container error: %v", err)
		}
		if err := storage.RemoveContainer(ctx, e.Docker, containerID); err != nil {
			log.Printf("[ENGINE] Remove container error: %v", err)
		}
	}

	if volumeID != "" {
		storage.RemoveVolumeDir(volumeID)
	}

	database.Exec(`UPDATE volumes SET status = 'deleted' WHERE id = ?`, volumeID)
	database.Exec(`UPDATE workspaces SET status = 'stopped' WHERE id = ?`, workspaceID)

	log.Printf("[ENGINE] Workspace %s terminated", workspaceID)
	return nil
}

func (e *Engine) cleanupVolume(volumeID string) {
	database.Exec(`UPDATE volumes SET status = 'failed' WHERE id = ?`, volumeID)
	storage.RemoveVolumeDir(volumeID)
}
