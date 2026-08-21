package tests

import (
	"os"
	"strings"
	"testing"
)

func TestOpenAPIDocumentsGovernanceWriteAndRecoveryRoutes(t *testing.T) {
	payload, err := os.ReadFile("../api/openapi/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	document := string(payload)
	required := []string{
		"/tables/{id}/classification:",
		"/policies/{id}/submit:",
		"/approvals/{id}/decision:",
		"/batches/{id}/recover:",
		"/batches/{id}/rollback:",
	}
	for _, route := range required {
		if !strings.Contains(document, route) {
			t.Errorf("OpenAPI is missing %s", route)
		}
	}
}
