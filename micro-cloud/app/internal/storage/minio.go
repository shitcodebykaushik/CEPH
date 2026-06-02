package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioManager struct {
	Client    *minio.Client
	Endpoint  string
	AccessKey string
	SecretKey string
}

func NewMinioManager(endpoint, accessKey, secretKey string, useSSL bool) (*MinioManager, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	log.Printf("[MINIO] Connected to %s", endpoint)
	return &MinioManager{Client: client, Endpoint: endpoint, AccessKey: accessKey, SecretKey: secretKey}, nil
}

func (m *MinioManager) CreateBucket(ctx context.Context, name string) error {
	exists, err := m.Client.BucketExists(ctx, name)
	if err != nil {
		return fmt.Errorf("check bucket %s: %w", name, err)
	}
	if exists {
		log.Printf("[MINIO] Bucket %q already exists", name)
		return nil
	}
	if err := m.Client.MakeBucket(ctx, name, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket %s: %w", name, err)
	}
	log.Printf("[MINIO] Created bucket %q", name)
	return nil
}

func (m *MinioManager) RemoveBucket(ctx context.Context, name string) error {
	if err := m.Client.RemoveBucket(ctx, name); err != nil {
		return fmt.Errorf("remove bucket %s: %w", name, err)
	}
	log.Printf("[MINIO] Removed bucket %q", name)
	return nil
}
