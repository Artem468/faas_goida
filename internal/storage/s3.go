package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	neturl "net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client        *minio.Client
	presignClient *minio.Client
	bucket        string
	publicBase    string
}

type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	Bucket          string
	PublicBaseURL   string
}

func NewS3(cfg Config) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, err
	}

	publicBase := strings.TrimRight(cfg.PublicBaseURL, "/")
	if publicBase == "" {
		scheme := "http"
		if cfg.UseSSL {
			scheme = "https"
		}
		publicBase = fmt.Sprintf("%s://%s", scheme, strings.TrimRight(cfg.Endpoint, "/"))
	}

	parsedPublic, err := neturl.Parse(publicBase)
	if err != nil {
		return nil, err
	}
	presignEndpoint := parsedPublic.Host
	presignSecure := strings.EqualFold(parsedPublic.Scheme, "https")
	presignClient, err := minio.New(presignEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: presignSecure,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, err
	}

	return &S3{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		publicBase:    publicBase,
	}, nil
}

func (s *S3) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func (s *S3) Upload(ctx context.Context, projectID int64, fileName string, body io.Reader, size int64, contentType string) (string, string, error) {
	key, err := objectKey(projectID, fileName)
	if err != nil {
		return "", "", err
	}

	_, err = s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", "", err
	}

	return key, s.buildURL(key), nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	u, err := s.presignClient.PresignedGetObject(ctx, s.bucket, key, ttl, neturl.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *S3) buildURL(key string) string {
	return fmt.Sprintf("%s/%s/%s", s.publicBase, s.bucket, key)
}

func objectKey(projectID int64, fileName string) (string, error) {
	ext := filepath.Ext(fileName)
	token, err := randomHex(8)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("projects/%d/%s%s", projectID, token, ext), nil
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
