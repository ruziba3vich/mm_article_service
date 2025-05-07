package storage

import (
	"bytes"
	"context"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/ruziba3vich/mm_article_service/internal/repos"
)

// minioStorage implements MinIOStorage
type MinioStorage struct {
	client     *minio.Client
	bucketName string
	urlExpiry  int64 // Seconds for presigned URL expiry
}

// NewMinIOStorage initializes a MinIO client
func NewMinIOStorage(endpoint, accessKey, secretKey, bucketName string, urlExpiry int64) (repos.MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	exists, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}
	return &MinioStorage{
		client:     client,
		bucketName: bucketName,
		urlExpiry:  urlExpiry,
	}, nil
}

// CreateFile stores a file in MinIO
func (s *MinioStorage) CreateFile(ctx context.Context, fileName string, fileContent []byte) (string, string, error) {
	ext := filepath.Ext(fileName)
	generatedName := uuid.New().String() + ext
	_, err := s.client.PutObject(ctx, s.bucketName, generatedName, bytes.NewReader(fileContent), int64(len(fileContent)), minio.PutObjectOptions{})
	if err != nil {
		return "", "", err
	}
	url, err := s.GetFileURL(ctx, generatedName)
	if err != nil {
		return "", "", err
	}
	return generatedName, url, nil
}

// DeleteFile removes a file from MinIO
func (s *MinioStorage) DeleteFile(ctx context.Context, fileName string) error {
	return s.client.RemoveObject(ctx, s.bucketName, fileName, minio.RemoveObjectOptions{})
}

// GetFileURL generates a temporary URL for a file
func (s *MinioStorage) GetFileURL(ctx context.Context, fileName string) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucketName, fileName, time.Duration(s.urlExpiry)*time.Second, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
