package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func GetMountBasePath() string {
	base := os.Getenv("VOLUME_MOUNT_BASE")
	if base == "" {
		base = "/tmp/micro-cloud/volumes"
	}
	return base
}

func VolumeDirPath(volumeID string) string {
	return filepath.Join(GetMountBasePath(), volumeID)
}

func CreateVolumeDir(volumeID string) error {
	path := VolumeDirPath(volumeID)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	log.Printf("[VOLUME] Created directory %s", path)
	return nil
}

func RemoveVolumeDir(volumeID string) error {
	path := VolumeDirPath(volumeID)
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	log.Printf("[VOLUME] Removed directory %s", path)
	return nil
}
