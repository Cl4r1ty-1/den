package storage

import (
    "context"
    "fmt"
    "net/url"
    "os"
    "time"

    minio "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type R2Client struct {
    client   *minio.Client
    bucket   string
}

func NewR2ClientFromEnv() (*R2Client, error) {
    accountID := os.Getenv("R2_ACCOUNT_ID")
    accessKey := os.Getenv("R2_ACCESS_KEY_ID")
    secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
    bucket := os.Getenv("R2_BUCKET")
    if accountID == "" || accessKey == "" || secretKey == "" || bucket == "" {
        return nil, fmt.Errorf("missing R2 configuration in environment")
    }
    endpoint := fmt.Sprintf("%s.r2.cloudflarestorage.com", accountID)
    cli, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: true,
        Region: "auto",
        BucketLookup: minio.BucketLookupPath,
    })
    if err != nil {
        return nil, err
    }
    return &R2Client{client: cli, bucket: bucket}, nil
}

func (s *R2Client) PresignedPut(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
    if expires <= 0 { expires = 2 * time.Hour }
    u, err := s.client.PresignedPutObject(ctx, s.bucket, objectKey, expires)
    if err != nil { return "", err }
    return u.String(), nil
}

func (s *R2Client) PresignedGet(ctx context.Context, objectKey string, expires time.Duration) (string, error) {
    if expires <= 0 { expires = 24 * time.Hour }
    u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expires, url.Values{})
    if err != nil { return "", err }
    return u.String(), nil
}

func (s *R2Client) DeleteObject(ctx context.Context, objectKey string) error {
    return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

