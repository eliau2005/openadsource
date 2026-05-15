package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakePresigner struct {
	gotKey string
	gotTTL time.Duration
	url    string
	err    error
}

func (f *fakePresigner) Presign(_ context.Context, key string, ttl time.Duration) (string, error) {
	f.gotKey = key
	f.gotTTL = ttl
	return f.url, f.err
}

func TestResolver_ExternalURL_Passthrough(t *testing.T) {
	r := &resolver{}
	url, err := r.ResolveMediaURL(context.Background(), SourceExternalURL, "https://cdn.example.com/clip.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://cdn.example.com/clip.mp4" {
		t.Errorf("url: want passthrough, got %q", url)
	}
}

func TestResolver_InternalS3_PublicBase(t *testing.T) {
	r := &resolver{publicBaseURL: "http://localhost:9000/openadsource"}
	url, err := r.ResolveMediaURL(context.Background(), SourceInternalS3, "seed/clip.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "http://localhost:9000/openadsource/seed/clip.mp4"; url != want {
		t.Errorf("url: want %q, got %q", want, url)
	}
}

func TestResolver_InternalS3_PublicBase_TrimsSlashes(t *testing.T) {
	r := &resolver{publicBaseURL: "http://localhost:9000/openadsource"}
	url, err := r.ResolveMediaURL(context.Background(), SourceInternalS3, "/seed/clip.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "http://localhost:9000/openadsource/seed/clip.mp4"; url != want {
		t.Errorf("url: want %q, got %q", want, url)
	}
}

func TestResolver_InternalS3_Presign(t *testing.T) {
	fake := &fakePresigner{url: "https://signed.example/clip.mp4?sig=xyz"}
	r := &resolver{presignFn: fake, presignTTL: 90 * time.Minute}
	url, err := r.ResolveMediaURL(context.Background(), SourceInternalS3, "seed/clip.mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != fake.url {
		t.Errorf("url: want %q, got %q", fake.url, url)
	}
	if fake.gotKey != "seed/clip.mp4" {
		t.Errorf("presigner saw key %q", fake.gotKey)
	}
	if fake.gotTTL != 90*time.Minute {
		t.Errorf("presigner saw ttl %v", fake.gotTTL)
	}
}

func TestResolver_InternalS3_NoBackend_Errors(t *testing.T) {
	r := &resolver{}
	_, err := r.ResolveMediaURL(context.Background(), SourceInternalS3, "seed/clip.mp4")
	if err == nil {
		t.Fatal("expected error when internal_s3 ad and no backend, got nil")
	}
}

func TestResolver_PresignError_Propagates(t *testing.T) {
	fake := &fakePresigner{err: errors.New("boom")}
	r := &resolver{presignFn: fake}
	_, err := r.ResolveMediaURL(context.Background(), SourceInternalS3, "k")
	if err == nil {
		t.Fatal("expected propagated error, got nil")
	}
}

func TestResolver_UnknownSource_Errors(t *testing.T) {
	r := &resolver{}
	_, err := r.ResolveMediaURL(context.Background(), "magic", "x")
	if err == nil {
		t.Fatal("expected error for unknown source, got nil")
	}
}
