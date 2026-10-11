package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Local stores objects as files under a root directory (development and tests).
// Content types are kept in a sidecar file next to each object.
type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("storage root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &Local{root: abs}, nil
}

const metaSuffix = ".meta"

func (l *Local) path(key string) (string, error) {
	if err := ValidateKey(key); err != nil || strings.HasSuffix(key, metaSuffix) {
		return "", ErrInvalidKey
	}
	p := filepath.Join(l.root, filepath.FromSlash(key))
	if !strings.HasPrefix(p, l.root+string(filepath.Separator)) {
		return "", ErrInvalidKey
	}
	return p, nil
}

func (l *Local) Put(_ context.Context, key string, r io.Reader, contentType string) (Object, error) {
	p, err := l.path(key)
	if err != nil {
		return Object{}, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return Object{}, fmt.Errorf("storage put: %w", err)
	}
	// Write to a temp file and rename, so readers never see a half-written object.
	tmp, err := os.CreateTemp(filepath.Dir(p), ".upload-*")
	if err != nil {
		return Object{}, fmt.Errorf("storage put: %w", err)
	}
	defer os.Remove(tmp.Name())
	size, err := io.Copy(tmp, r)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return Object{}, fmt.Errorf("storage put: %w", err)
	}
	if err := os.WriteFile(p+metaSuffix, []byte(contentType), 0o640); err != nil {
		return Object{}, fmt.Errorf("storage put: %w", err)
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return Object{}, fmt.Errorf("storage put: %w", err)
	}
	return Object{Key: key, Size: size, ContentType: contentType}, nil
}

func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, Object, error) {
	p, err := l.path(key)
	if err != nil {
		return nil, Object{}, err
	}
	f, err := os.Open(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, Object{}, ErrNotFound
	}
	if err != nil {
		return nil, Object{}, fmt.Errorf("storage open: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, Object{}, fmt.Errorf("storage open: %w", err)
	}
	ct, _ := os.ReadFile(p + metaSuffix)
	return f, Object{Key: key, Size: info.Size(), ContentType: string(ct)}, nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	for _, f := range []string{p, p + metaSuffix} {
		if err := os.Remove(f); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("storage delete: %w", err)
		}
	}
	return nil
}
