package config

import (
	"github.com/wyw14/cry052/internal/platform"
	"path/filepath"
	"testing"
)

func TestWorkspaceBoundaryRejectsSecretsCachesAndRuntimeArtifacts(t *testing.T) {
	root := t.TempDir()
	bad := []string{filepath.Join(root, ".git", "secrets"), filepath.Join(root, "web", "node_modules", "uploads"), filepath.Join(root, "dist", "runtime"), filepath.Join(root, ".cache", "attachments")}
	for _, directory := range bad {
		boundary := WorkspaceBoundary{Root: root, AttachmentDirectory: directory, SecretReferences: []string{"hash/default"}}
		if err := boundary.Validate(); err == nil {
			t.Errorf("unsafe artifact directory accepted: %s", directory)
		}
	}
	policy := platform.SecretReferencePolicy{Prefix: "hash/", MinNameLength: 3}
	for _, reference := range []string{"../token", "hash/../../token", "password=plain"} {
		if err := policy.Validate(reference); err == nil {
			t.Errorf("unsafe secret reference accepted: %q", reference)
		}
	}
	safe := WorkspaceBoundary{Root: root, AttachmentDirectory: filepath.Join(root, "data", "attachments"), SecretReferences: []string{"hash/default"}}
	if err := safe.Validate(); err != nil {
		t.Fatalf("safe offline boundary rejected: %v", err)
	}
}
