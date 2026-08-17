package blobstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DefaultMaxSize int64 = 10 << 20

var (
	ErrUnsupportedType = errors.New("unsupported attachment content type")
	ErrTooLarge        = errors.New("attachment exceeds size limit")
)

type Saved struct {
	Path   string
	SHA256 string
	Size   int64
}

type Local struct {
	root       string
	maxSize    int64
	contentSet map[string]struct{}
}

func NewLocal(root string, maxSize int64) (*Local, error) {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &Local{root: abs, maxSize: maxSize, contentSet: map[string]struct{}{
		"image/jpeg": {}, "image/png": {}, "image/webp": {}, "application/pdf": {},
	}}, nil
}

func (s *Local) Save(ctx context.Context, id, contentType string, source io.Reader) (Saved, error) {
	if _, ok := s.contentSet[strings.ToLower(contentType)]; !ok {
		return Saved{}, ErrUnsupportedType
	}
	name := filepath.Base(id)
	if name == "." || name == "" || name != id {
		return Saved{}, fmt.Errorf("invalid blob identifier")
	}
	path := filepath.Join(s.root, name)
	if !strings.HasPrefix(path, s.root+string(os.PathSeparator)) {
		return Saved{}, fmt.Errorf("blob path escapes storage root")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return Saved{}, err
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()
	hash := sha256.New()
	reader := io.LimitReader(source, s.maxSize+1)
	written, err := io.Copy(io.MultiWriter(file, hash), &contextReader{ctx: ctx, reader: reader})
	if err != nil {
		return Saved{}, err
	}
	if written > s.maxSize {
		return Saved{}, ErrTooLarge
	}
	if err := file.Sync(); err != nil {
		return Saved{}, err
	}
	remove = false
	return Saved{Path: path, SHA256: hex.EncodeToString(hash.Sum(nil)), Size: written}, nil
}

func (s *Local) Open(ctx context.Context, id string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	name := filepath.Base(id)
	if name != id || name == "." || name == "" {
		return nil, fmt.Errorf("invalid blob identifier")
	}
	return os.Open(filepath.Join(s.root, name))
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
