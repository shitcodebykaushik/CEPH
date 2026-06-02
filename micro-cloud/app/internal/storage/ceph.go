package storage

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ceph/go-ceph/rados"
	"github.com/ceph/go-ceph/rbd"
)

type CephManager struct {
	Conn *rados.Conn
}

func NewCephManager(configPath string) (*CephManager, error) {
	confFile := configPath
	if confFile == "" {
		confFile = "/etc/ceph/ceph.conf"
	}

	conn, err := rados.NewConn()
	if err != nil {
		return nil, fmt.Errorf("rados new conn: %w", err)
	}

	if err := conn.ReadConfigFile(confFile); err != nil {
		return nil, fmt.Errorf("read config %s: %w", confFile, err)
	}

	conn.SetConfigOption("rados_mon_op_timeout", "15")
	conn.SetConfigOption("rados_osd_op_timeout", "30")

	if err := conn.Connect(); err != nil {
		return nil, fmt.Errorf("connect ceph: %w", err)
	}

	fsid, _ := conn.GetFSID()
	log.Printf("[CEPH] Connected, FSID: %s", fsid)

	if err := ensurePool(conn, "rbd"); err != nil {
		return nil, fmt.Errorf("ensure pool: %w", err)
	}

	return &CephManager{Conn: conn}, nil
}

func (m *CephManager) Close() {
	m.Conn.Shutdown()
}

func ensurePool(conn *rados.Conn, name string) error {
	pools, err := conn.ListPools()
	if err != nil {
		return fmt.Errorf("list pools: %w", err)
	}
	for _, p := range pools {
		if p == name {
			log.Printf("[CEPH] Pool %q already exists", name)
			return nil
		}
	}
	if err := conn.MakePool(name); err != nil {
		return fmt.Errorf("create pool %s: %w", name, err)
	}
	log.Printf("[CEPH] Created pool %q", name)
	return nil
}

func (m *CephManager) CreateRBDImage(pool, name string, sizeMB int) error {
	ioctx, err := m.Conn.OpenIOContext(pool)
	if err != nil {
		return fmt.Errorf("open ioctx %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	sizeBytes := uint64(sizeMB) * 1024 * 1024
	if _, err := rbd.Create(ioctx, name, sizeBytes, 22); err != nil {
		return fmt.Errorf("rbd create %s: %w", name, err)
	}
	log.Printf("[CEPH] RBD image %s/%s created (%d MB)", pool, name, sizeMB)
	return nil
}

func (m *CephManager) RemoveRBDImage(pool, name string) error {
	ioctx, err := m.Conn.OpenIOContext(pool)
	if err != nil {
		return fmt.Errorf("open ioctx %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	img := rbd.GetImage(ioctx, name)
	if err := img.Remove(); err != nil {
		return fmt.Errorf("rbd remove %s/%s: %w", pool, name, err)
	}
	log.Printf("[CEPH] RBD image %s/%s removed", pool, name)
	return nil
}

func (m *CephManager) GetImageSize(pool, name string) (uint64, error) {
	ioctx, err := m.Conn.OpenIOContext(pool)
	if err != nil {
		return 0, fmt.Errorf("open ioctx %s: %w", pool, err)
	}
	defer ioctx.Destroy()

	img := rbd.GetImage(ioctx, name)
	info, err := img.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", name, err)
	}
	return info.Size, nil
}

func MapRBDDevice(pool, image string) (string, error) {
	cmd := exec.Command("rbd-nbd", "map",
		fmt.Sprintf("%s/%s", pool, image),
		"--id", "admin",
		"--conf", "/etc/ceph/ceph.conf",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("rbd-nbd map failed: %s: %w", string(output), err)
	}
	devPath := strings.TrimSpace(string(output))
	log.Printf("[CEPH] Mapped %s/%s -> %s", pool, image, devPath)
	return devPath, nil
}

func UnmapRBDDevice(devicePath string) error {
	cmd := exec.Command("rbd-nbd", "unmap", devicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rbd-nbd unmap failed: %s: %w", string(output), err)
	}
	log.Printf("[CEPH] Unmapped device %s", devicePath)
	return nil
}



func FormatDevice(devicePath string) error {
	cmd := exec.Command("mkfs.ext4", "-F", devicePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkfs.ext4 failed: %s: %w", string(output), err)
	}
	log.Printf("[CEPH] Formatted %s with ext4", devicePath)
	return nil
}

func MountDevice(devicePath, mountPoint string) error {
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("mkdir mount point %s: %w", mountPoint, err)
	}

	cmd := exec.Command("mount", devicePath, mountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount %s -> %s failed: %s: %w", devicePath, mountPoint, string(output), err)
	}
	log.Printf("[CEPH] Mounted %s at %s", devicePath, mountPoint)
	return nil
}

func UnmountDevice(mountPoint string) error {
	cmd := exec.Command("umount", mountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("umount %s failed: %s: %w", mountPoint, string(output), err)
	}
	return nil
}

func GetMountBasePath() string {
	base := os.Getenv("VOLUME_MOUNT_BASE")
	if base == "" {
		base = "/mnt/micro-cloud/volumes"
	}
	return base
}

func VolumeMountPath(volumeID string) string {
	return filepath.Join(GetMountBasePath(), volumeID)
}
