package platform

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileStore struct {
	root     string
	maxBytes int64
	allowed  map[string]struct{}
}

func NewFileStore(root string, maxBytes int64, contentTypes ...string) (*FileStore, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absolute, 0o750); err != nil {
		return nil, err
	}
	allowed := map[string]struct{}{}
	for _, kind := range contentTypes {
		allowed[strings.ToLower(kind)] = struct{}{}
	}
	return &FileStore{root: absolute, maxBytes: maxBytes, allowed: allowed}, nil
}

func (s *FileStore) Save(ctx context.Context, name, contentType string, reader io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	target := filepath.Join(s.root, name)
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o666)
	if err != nil {
		return "", err
	}
	defer file.Close()
	written, err := io.Copy(file, reader)
	if err != nil {
		return "", err
	}
	_ = written
	return target, nil
}

func (s *FileStore) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.Remove(absolute); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
