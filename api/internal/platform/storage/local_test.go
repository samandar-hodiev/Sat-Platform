package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

var _ ObjectStorage = (*Local)(nil)

func TestValidateKey(t *testing.T) {
	good := []string{"avatars/u1/a.webp", "x", "lessons/2026/img_1.png"}
	bad := []string{"", "/etc/passwd", "../secret", "a/../b", "a//b", "a/./b", ".hidden", "a b", "avatars/ü.png"}
	for _, k := range good {
		if err := ValidateKey(k); err != nil {
			t.Errorf("%q rejected: %v", k, err)
		}
	}
	for _, k := range bad {
		if err := ValidateKey(k); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("%q accepted", k)
		}
	}
}

func TestLocalRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	obj, err := s.Put(ctx, "avatars/u1/a.webp", strings.NewReader("image-bytes"), "image/webp")
	if err != nil {
		t.Fatal(err)
	}
	if obj.Size != int64(len("image-bytes")) {
		t.Errorf("size = %d", obj.Size)
	}

	rc, got, err := s.Open(ctx, "avatars/u1/a.webp")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(rc)
	rc.Close()
	if string(body) != "image-bytes" || got.ContentType != "image/webp" {
		t.Errorf("got %q / %q", body, got.ContentType)
	}

	if err := s.Delete(ctx, "avatars/u1/a.webp"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Open(ctx, "avatars/u1/a.webp"); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := s.Delete(ctx, "avatars/u1/a.webp"); err != nil {
		t.Errorf("deleting a missing object must succeed: %v", err)
	}
}

func TestLocalRejectsTraversal(t *testing.T) {
	s, _ := NewLocal(t.TempDir())
	if _, err := s.Put(context.Background(), "../escape.txt", strings.NewReader("x"), "text/plain"); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("traversal accepted: %v", err)
	}
	if _, err := s.Put(context.Background(), "a/b.meta", strings.NewReader("x"), "text/plain"); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("sidecar key accepted: %v", err)
	}
}
