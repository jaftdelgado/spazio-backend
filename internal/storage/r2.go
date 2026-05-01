package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/jaftdelgado/spazio-backend/internal/config"
)

type R2Client struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	publicURL string
}

func NewR2Client(cfg appconfig.R2Config) (*R2Client, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	r2cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load r2 config: %w", err)
	}

	client := s3.NewFromConfig(r2cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &R2Client{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    cfg.BucketName,
		publicURL: cfg.PublicURL,
	}, nil
}

// PresignGetURL generates a temporary signed URL to read an object.
func (r *R2Client) PresignGetURL(ctx context.Context, storageKey string, ttl time.Duration) (string, error) {
	resp, err := r.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(storageKey),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign get url: %w", err)
	}
	return resp.URL, nil
}

// PublicURL returns the public URL for an object if a public domain is configured,
// otherwise falls back to a presigned URL with a 1-hour TTL.
func (r *R2Client) PublicURL(ctx context.Context, storageKey string) (string, error) {
	if r.publicURL != "" {
		return fmt.Sprintf("%s/%s", r.publicURL, storageKey), nil
	}
	return r.PresignGetURL(ctx, storageKey, time.Hour)
}

// Upload uploads a file to R2.
func (r *R2Client) Upload(ctx context.Context, storageKey string, contentType string, body io.Reader) error {
	log.Printf("[R2 Upload] bucket=%q key=%q contentType=%q", r.bucket, storageKey, contentType)

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(storageKey),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Printf("[R2 Upload] error: %v", err)
		return fmt.Errorf("upload to r2: %w", err)
	}
	log.Printf("[R2 Upload] success: %q", storageKey)
	return nil
}

// Delete removes an object from R2.
func (r *R2Client) Delete(ctx context.Context, storageKey string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(storageKey),
	})
	if err != nil {
		return fmt.Errorf("delete from r2: %w", err)
	}
	return nil
}
