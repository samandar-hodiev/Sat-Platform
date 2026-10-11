// Package storage is the file storage abstraction for uploads (avatars, wallpapers,
// lesson and question images). Modules depend on ObjectStorage, never on a concrete
// backend, so local disk (development) can be replaced by S3-compatible storage later
// without touching module code.
//
// Storage keeps bytes only. Validating uploads (size, content sniffing, allowed types)
// is the uploading module's job, before Put.
package storage

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
)

var (
	ErrNotFound   = errors.New("storage: object not found")
	ErrInvalidKey = errors.New("storage: invalid key")
)

// Object describes a stored file.
type Object struct {
	Key         string
	Size        int64
	ContentType string
}

type ObjectStorage interface {
	// Put stores r under key, replacing any existing object.
	Put(ctx context.Context, key string, r io.Reader, contentType string) (Object, error)
	// Open returns the object's content; the caller closes it. ErrNotFound if missing.
	Open(ctx context.Context, key string) (io.ReadCloser, Object, error)
	// Delete removes the object. Deleting a missing object is not an error.
	Delete(ctx context.Context, key string) error
}

// Keys are slash-separated paths of safe segments, e.g. "avatars/<user-id>/<uuid>.webp".
// Modules generate them; user-supplied file names never become keys.
var keySegment = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ValidateKey rejects empty keys, absolute paths, traversal (".."), and unexpected characters.
func ValidateKey(key string) error {
	if key == "" || len(key) > 512 || strings.HasPrefix(key, "/") {
		return ErrInvalidKey
	}
	for _, seg := range strings.Split(key, "/") {
		if seg == "." || seg == ".." || !keySegment.MatchString(seg) {
			return ErrInvalidKey
		}
	}
	return nil
}
