// Package storage owns the logic that turns an `ads` row into a playable
// media URL. Phase 1 supports two backends:
//   - external_url: the ad row's media_url is already a fully-qualified public
//     URL (Bunny.net, Cloudflare Stream, archive.org, etc.) — passthrough.
//   - internal_s3: the ad row's media_url is an S3 object key relative to the
//     configured bucket. If S3_PUBLIC_BASE_URL is set we serve the object via
//     a static public base; otherwise we presign a short-lived GET URL.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/eliau2005/openadsource/server/internal/config"
)

// S3Client wraps the AWS SDK v2 S3 client with the bits we actually use:
// presigning GETs (for the resolver) and putting objects (for the seed
// script). All endpoint / region / path-style options are env-driven so the
// same client works against AWS S3, MinIO, R2, Bunny Edge Storage, Wasabi, B2.
type S3Client struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

// NewS3Client constructs an S3 client from environment-driven config. Returns
// (nil, nil) silently when S3 isn't configured (Configured() == false) so
// callers can fall back to passthrough resolution.
func NewS3Client(ctx context.Context, cfg config.Config) (*S3Client, error) {
	if !cfg.S3Configured() {
		return nil, nil
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3AccessKeyID, cfg.S3SecretAccessKey, "",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		o.UsePathStyle = cfg.S3ForcePathStyle
	})

	return &S3Client{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    cfg.S3Bucket,
	}, nil
}

// Presign returns a short-lived URL the player can fetch the object from. ttl
// caps how long the URL stays valid; for VAST responses an hour is plenty
// (the player consumes immediately).
func (c *S3Client) Presign(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if c == nil {
		return "", errors.New("s3 client is nil")
	}
	req, err := c.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, func(o *s3.PresignOptions) {
		o.Expires = ttl
	})
	if err != nil {
		return "", fmt.Errorf("presign get %s: %w", key, err)
	}
	return req.URL, nil
}

// PutObject uploads bytes to the configured bucket under key. Used by the
// seed script to stage a sample MP4 in MinIO so the resolver can serve it.
func (c *S3Client) PutObject(ctx context.Context, key string, body io.Reader, contentType string) error {
	if c == nil {
		return errors.New("s3 client is nil")
	}
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

// Bucket returns the bucket name the client was constructed with. Useful
// for logging / diagnostics.
func (c *S3Client) Bucket() string {
	if c == nil {
		return ""
	}
	return c.bucket
}
