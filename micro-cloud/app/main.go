package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ceph/go-ceph/rados"
	"github.com/docker/docker/client"
)

func main() {
	fmt.Println("=== Micro-Cloud Orchestrator Starting ===")

	// ── 1. Connect to Ceph ──────────────────────────────────────
	cephConn, err := connectCeph()
	if err != nil {
		log.Fatalf("[CEPH] Connection failed: %v", err)
	}
	defer cephConn.Shutdown()
	fmt.Println("[CEPH] ✓ Connected to cluster successfully")

	// Print cluster FSID to prove real connection
	fsid, err := cephConn.GetFSID()
	if err != nil {
		log.Fatalf("[CEPH] Could not get FSID: %v", err)
	}
	fmt.Printf("[CEPH] ✓ Cluster FSID: %s\n", fsid)

	// ── 2. Connect to Docker ────────────────────────────────────
	dockerClient, err := connectDocker()
	if err != nil {
		log.Fatalf("[DOCKER] Connection failed: %v", err)
	}
	defer dockerClient.Close()
	fmt.Println("[DOCKER] ✓ Connected to Docker daemon successfully")

	// Print Docker version to prove real connection
	version, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		log.Fatalf("[DOCKER] Could not get server version: %v", err)
	}
	fmt.Printf("[DOCKER] ✓ Docker version: %s (API: %s)\n", version.Version, version.APIVersion)

	fmt.Println("\n=== All systems connected. Ready for Milestone 3. ===")
}

// connectCeph reads /etc/ceph/ceph.conf and opens a connection
// to the Ceph cluster using the admin keyring.
func connectCeph() (*rados.Conn, error) {
	confFile := os.Getenv("CEPH_CONF")
	if confFile == "" {
		confFile = "/etc/ceph/ceph.conf"
	}

	// NewConn creates a handle using the "client.admin" user
	conn, err := rados.NewConn()
	if err != nil {
		return nil, fmt.Errorf("creating rados conn: %w", err)
	}

	// Read the ceph.conf file for mon address, cluster name, etc.
	if err := conn.ReadConfigFile(confFile); err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", confFile, err)
	}

	// Actually open the connection to the monitor
	if err := conn.Connect(); err != nil {
		return nil, fmt.Errorf("connecting to cluster: %w", err)
	}

	return conn, nil
}

// connectDocker creates a Docker client that talks to the host
// daemon via the /var/run/docker.sock volume we mounted.
func connectDocker() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(), // auto-negotiate API version
	)
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	return cli, nil
}
