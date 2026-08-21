package platform

import (
	"context"
	"fmt"
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
	if _, ok := s.allowed[strings.ToLower(contentType)]; !ok {
		return "", fmt.Errorf("content type is not allowed")
	}
	base := filepath.Base(name)
	if base == "." || base == "" || base != name {
		return "", fmt.Errorf("unsafe file name")
	}
	target := filepath.Join(s.root, base)
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(target)
		}
	}()
	written, err := io.Copy(file, io.LimitReader(reader, s.maxBytes+1))
	if err != nil {
		return "", err
	}
	if written > s.maxBytes {
		return "", fmt.Errorf("file exceeds size limit")
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	ok = true
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
	relative, err := filepath.Rel(s.root, absolute)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("attachment path is outside the controlled root")
	}
	if err := os.Remove(absolute); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
