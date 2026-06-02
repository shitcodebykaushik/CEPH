package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
	"github.com/shiiit/micro-cloud/internal/database"
	"github.com/shiiit/micro-cloud/internal/storage"
)

type Engine struct {
	Ceph   *storage.CephManager
	Docker *client.Client
}

func NewEngine(cm *storage.CephManager, dc *client.Client) *Engine {
	return &Engine{
		Ceph:   cm,
		Docker: dc,
	}
}

func (e *Engine) CreateVolume(tenantID, volumeID, imageName, pool string, sizeMB int) error {
	if err := e.Ceph.CreateRBDImage(pool, imageName, sizeMB); err != nil {
		return fmt.Errorf("create rbd image: %w", err)
	}

	if _, err := database.Exec(
		`UPDATE volumes SET status = 'available' WHERE id = ?`,
		volumeID,
	); err != nil {
		return fmt.Errorf("update volume status: %w", err)
	}

	log.Printf("[ENGINE] Volume %s ready (%d MB, pool=%s, image=%s)", volumeID, sizeMB, pool, imageName)
	return nil
}

func (e *Engine) LaunchWorkspace(workspaceID, volumeID, tenantID, imageName, pool string, sizeMB int, containerImage string) error {
	log.Printf("[ENGINE] Launching workspace %s for tenant %s", workspaceID, tenantID)

	if err := e.Ceph.CreateRBDImage(pool, imageName, sizeMB); err != nil {
		database.Exec(`UPDATE volumes SET status = 'failed' WHERE id = ?`, volumeID)
		return fmt.Errorf("create rbd image: %w", err)
	}

	mountPath := storage.VolumeMountPath(volumeID)

	devPath, err := storage.MapRBDDevice(pool, imageName)
	if err != nil {
		log.Printf("[ENGINE] RBD map failed, using directory fallback: %v", err)
		if err := os.MkdirAll(mountPath, 0755); err != nil {
			e.cleanupVolume(volumeID, pool, imageName, "")
			return fmt.Errorf("mkdir fallback: %w", err)
		}
		database.Exec(
			`UPDATE volumes SET status = 'attached', mount_path = ? WHERE id = ?`,
			mountPath, volumeID,
		)
	} else {
		devPath = sanitizeDevicePath(devPath)

		if err := storage.FormatDevice(devPath); err != nil {
			log.Printf("[ENGINE] Format failed: %v", err)
			e.cleanupVolume(volumeID, pool, imageName, devPath)
			return fmt.Errorf("format device: %w", err)
		}

		if err := storage.MountDevice(devPath, mountPath); err != nil {
			log.Printf("[ENGINE] Mount failed: %v", err)
			e.cleanupVolume(volumeID, pool, imageName, devPath)
			return fmt.Errorf("mount device: %w", err)
		}

		database.Exec(
			`UPDATE volumes SET status = 'attached', device_path = ?, mount_path = ? WHERE id = ?`,
			devPath, mountPath, volumeID,
		)
		log.Printf("[ENGINE] Volume %s mapped at %s -> %s", volumeID, devPath, mountPath)
	}

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
		if devPath != "" {
			storage.UnmountDevice(mountPath)
			storage.UnmapRBDDevice(devPath)
		}
		e.cleanupVolume(volumeID, pool, imageName, devPath)
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

	var volumeMountPath, devicePath, poolName, imageName string
	row := database.QueryRow(
		`SELECT COALESCE(mount_path,''), COALESCE(device_path,''), pool_name, image_name FROM volumes WHERE id = ?`,
		volumeID,
	)
	row.Scan(&volumeMountPath, &devicePath, &poolName, &imageName)

	if volumeMountPath != "" {
		storage.UnmountDevice(volumeMountPath)
	}
	if devicePath != "" {
		storage.UnmapRBDDevice(devicePath)
	}

	if poolName != "" && imageName != "" {
		if err := e.Ceph.RemoveRBDImage(poolName, imageName); err != nil {
			log.Printf("[ENGINE] Remove RBD image error: %v", err)
		}
	}

	database.Exec(`UPDATE volumes SET status = 'deleted' WHERE id = ?`, volumeID)
	database.Exec(`UPDATE workspaces SET status = 'stopped' WHERE id = ?`, workspaceID)

	log.Printf("[ENGINE] Workspace %s terminated", workspaceID)
	return nil
}

func (e *Engine) cleanupVolume(volumeID, pool, imageName, devPath string) {
	database.Exec(`UPDATE volumes SET status = 'failed' WHERE id = ?`, volumeID)
	if devPath != "" {
		if err := storage.UnmapRBDDevice(devPath); err != nil {
			log.Printf("[CLEANUP] Unmap error: %v", err)
		}
	}
	if pool != "" && imageName != "" {
		if err := e.Ceph.RemoveRBDImage(pool, imageName); err != nil {
			log.Printf("[CLEANUP] Remove image error: %v", err)
		}
	}
}

func sanitizeDevicePath(raw string) string {
	path := raw
	if path == "" {
		return path
	}
	path = filepath.Clean(path)

	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved
	}

	return path
}
