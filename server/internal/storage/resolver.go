package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eliau2005/openadsource/server/internal/config"
	"github.com/eliau2005/openadsource/server/internal/db"
)

const (
	// SourceExternalURL is the media_source enum value for ads whose media_url
	// is already a fully-qualified public URL (CDN, archive.org, etc.).
	SourceExternalURL = "external_url"
	// SourceInternalS3 is the media_source enum value for ads whose media_url
	// is an S3 object key, served from the configured bucket.
	SourceInternalS3 = "internal_s3"

	// PresignTTL is how long a presigned URL stays valid. Players consume the
	// VAST response immediately, so a short window is fine; an hour is the
	// hold-time during which the same VAST response can be re-fetched without
	// regenerating signatures.
	PresignTTL = time.Hour
)

// presigner is the minimal interface Resolver needs from an S3Client. It is
// declared here (rather than depending on the concrete *S3Client) so tests
// can inject a fake without an AWS SDK roundtrip.
type presigner interface {
	Presign(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// Resolver turns a database row into a playable URL plus its MIME type. The
// MIME is returned alongside the URL because the storage backend (presigning
// vs public base URL) is independent of the underlying file's codec.
type Resolver interface {
	ResolveMediaURL(ctx context.Context, ad db.GetAdByIDRow) (url string, mime string, err error)
}

// New constructs the appropriate resolver. If S3 isn't configured (s3 == nil
// AND no public base URL), the resolver becomes a strict passthrough that
// will error on internal_s3 ads — which never happens under our seed flow
// because internal_s3 inserts only occur when S3 is reachable.
func New(cfg config.Config, s3 *S3Client) Resolver {
	return &resolver{
		s3:             s3,
		publicBaseURL:  strings.TrimRight(cfg.S3PublicBaseURL, "/"),
		presignTTL:     PresignTTL,
	}
}

type resolver struct {
	s3            *S3Client
	publicBaseURL string
	presignTTL    time.Duration
	// override hook used in tests to swap the presigner without a real S3
	// client. Production paths leave this nil and fall through to s3.
	presignFn presigner
}

func (r *resolver) ResolveMediaURL(ctx context.Context, ad db.GetAdByIDRow) (string, string, error) {
	switch ad.MediaSource {
	case SourceExternalURL:
		return ad.MediaUrl, ad.MediaMime, nil

	case SourceInternalS3:
		key := ad.MediaUrl
		if r.publicBaseURL != "" {
			return fmt.Sprintf("%s/%s", r.publicBaseURL, strings.TrimLeft(key, "/")), ad.MediaMime, nil
		}
		// Fall through to presigning.
		var p presigner = r.presignFn
		if p == nil {
			p = r.s3
		}
		if p == nil {
			return "", "", errors.New("internal_s3 ad but no S3 client or public base URL configured")
		}
		url, err := p.Presign(ctx, key, r.presignTTL)
		if err != nil {
			return "", "", err
		}
		return url, ad.MediaMime, nil

	default:
		return "", "", fmt.Errorf("unknown media_source %q", ad.MediaSource)
	}
}
