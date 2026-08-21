package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/wyw14/cry052/internal/config"
	"github.com/wyw14/cry052/internal/middleware"
)

func TestDiagnosisDeliveryConfigCollapsesDistinctSessionRoles(t *testing.T) {
	roles := map[string]string{
		"ADMIN_SESSION_TOKEN":    "data_admin",
		"REVIEWER_SESSION_TOKEN": "policy_reviewer",
		"AUDITOR_SESSION_TOKEN":  "auditor",
		"EXECUTOR_SESSION_TOKEN": "masking_executor",
	}
	t.Run("delivery sample separates role credentials", func(t *testing.T) {
		values := readEnvironmentExample(t, "../.env.example")
		seen := map[string]string{}
		for key, role := range roles {
			value := values[key]
			if value == "" {
				t.Errorf("delivery sample is missing %s for %s", key, role)
				continue
			}
			if previous := seen[value]; previous != "" {
				t.Errorf("delivery credential for %s duplicates %s", role, previous)
			}
			seen[value] = role
		}
	})
	t.Run("loader preserves four fixed identities", func(t *testing.T) {
		values := map[string]string{
			"ADMIN_SESSION_TOKEN": "admin-delivery", "REVIEWER_SESSION_TOKEN": "reviewer-delivery",
			"AUDITOR_SESSION_TOKEN": "auditor-delivery", "EXECUTOR_SESSION_TOKEN": "executor-delivery",
		}
		for key, value := range values {
			t.Setenv(key, value)
		}
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		auth := middleware.StaticAuthenticator{
			cfg.Sessions.Admin:    {ActorID: "admin", Role: roles["ADMIN_SESSION_TOKEN"]},
			cfg.Sessions.Reviewer: {ActorID: "reviewer", Role: roles["REVIEWER_SESSION_TOKEN"]},
			cfg.Sessions.Auditor:  {ActorID: "auditor", Role: roles["AUDITOR_SESSION_TOKEN"]},
			cfg.Sessions.Executor: {ActorID: "executor", Role: roles["EXECUTOR_SESSION_TOKEN"]},
		}
		if len(auth) != 4 {
			t.Fatalf("configured session roles collapsed to %d credential", len(auth))
		}
		for token, expectedRole := range map[string]string{
			values["ADMIN_SESSION_TOKEN"]: roles["ADMIN_SESSION_TOKEN"], values["REVIEWER_SESSION_TOKEN"]: roles["REVIEWER_SESSION_TOKEN"],
			values["AUDITOR_SESSION_TOKEN"]: roles["AUDITOR_SESSION_TOKEN"], values["EXECUTOR_SESSION_TOKEN"]: roles["EXECUTOR_SESSION_TOKEN"],
		} {
			if auth[token].Role != expectedRole {
				t.Errorf("token %q role=%q want=%q", token, auth[token].Role, expectedRole)
			}
		}
	})
}

func readEnvironmentExample(t *testing.T, path string) map[string]string {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(payload), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && key != "" {
			values[key] = value
		}
	}
	return values
}

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
