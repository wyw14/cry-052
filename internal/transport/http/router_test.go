package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/application"
	"github.com/wyw14/cry052/internal/domain"
	"github.com/wyw14/cry052/internal/middleware"
	"github.com/wyw14/cry052/internal/platform"
	"github.com/wyw14/cry052/internal/repository/memory"
	"github.com/wyw14/cry052/internal/service"
	"go.uber.org/zap"
)

type readyStore struct{ *memory.Store }

func (readyStore) Ping(context.Context) error { return nil }

type unavailableReadiness struct{ err error }

func (r unavailableReadiness) Ping(context.Context) error { return r.err }

type panicReadiness struct{}

func (panicReadiness) Ping(context.Context) error { panic("readiness exploded") }

type deadlineReadiness struct{}

func (deadlineReadiness) Ping(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(500 * time.Millisecond):
		return errors.New("probe outlived request deadline")
	}
}

func testRouter(t *testing.T) http.Handler {
	router, _ := testRouterWithStore(t)
	return router
}

func testRouterWithStore(t *testing.T) (http.Handler, *memory.Store) {
	t.Helper()
	store := readyStore{memory.New()}
	samples := platform.NewSampleDatabase()
	preview := service.NewPreviewEngine()
	vault := platform.NewSecretVault(map[string][]byte{"hash/default": []byte("0123456789abcdef")})
	registry, err := service.NewRegistry(service.MaskTransformer{}, service.ReplaceTransformer{}, service.GeneralizeTransformer{}, service.NewHashTransformer(vault), service.KeepTransformer{})
	if err != nil {
		t.Fatal(err)
	}
	processor := service.NewBatchProcessor(samples, store.Store, preview, 10, time.Now)
	files, err := platform.NewFileStore(t.TempDir(), 32, "text/csv", "application/json")
	if err != nil {
		t.Fatal(err)
	}
	app := application.New(application.Dependencies{Store: store.Store, Samples: samples, Notifier: platform.NewLocalNotifier(time.Now), Redactor: platform.NewRedactor(), Files: files, Callbacks: platform.NewLocalCallbackSink(), Scheduler: platform.NewLocalScheduler(), Compiler: service.NewCompiler(registry), Preview: preview, Processor: processor, Exporter: service.NewReportExporter()})
	auth := middleware.StaticAuthenticator{"test-admin-session": {ActorID: "admin", Role: "data_admin"}}
	return New(app, store, auth, zap.NewNop(), time.Second), store.Store
}

func TestTablesHTTPPaginatesAndPersistsClassificationUpdates(t *testing.T) {
	router, store := testRouterWithStore(t)
	now := time.Now().UTC()
	source, err := domain.NewDataSource("catalog-source", "Catalog source", domain.DataSourceSample, "secret/catalog", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateDataSource(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	table := domain.TableSchema{ID: "catalog-table", DataSourceID: source.ID, Schema: "sample", Name: "customers", Fingerprint: "catalog-v1", Version: 1, DiscoveredAt: now, Fields: []domain.FieldSchema{{Name: "email", DataType: "text", Sensitivity: domain.SensitivityInternal, Category: "contact", Scopes: []string{"support"}}}}
	if err := store.PutTable(context.Background(), table); err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v1/tables?data_source_id=catalog-source&page=1&size=1&sort=qualified_name&filter_category=contact", nil)
	list.Header.Set("Authorization", "Bearer test-admin-session")
	listed := httptest.NewRecorder()
	router.ServeHTTP(listed, list)
	if listed.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	var page domain.Page[domain.TableSchema]
	if err := json.Unmarshal(listed.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page=%+v", page)
	}

	table.Fields[0].Sensitivity = domain.SensitivityRestricted
	payload, _ := json.Marshal(map[string]any{"version": int64(1), "fields": table.Fields})
	update := httptest.NewRequest(http.MethodPatch, "/api/v1/tables/catalog-table/classification", bytes.NewReader(payload))
	update.Header.Set("Content-Type", "application/json")
	update.Header.Set("Authorization", "Bearer test-admin-session")
	updated := httptest.NewRecorder()
	router.ServeHTTP(updated, update)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", updated.Code, updated.Body.String())
	}
	persisted, err := store.GetTable(context.Background(), table.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Version != 2 || persisted.Fields[0].Sensitivity != domain.SensitivityRestricted {
		t.Fatalf("persisted=%+v", persisted)
	}
}

func TestCreateDataSourceHTTPValidatesAndRedactsContract(t *testing.T) {
	router := testRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/data-sources", bytes.NewBufferString(`{"name":"本地样例库","kind":"local_sample","connection_reference":"secret/demo"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer test-admin-session")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte("postgres://")) {
		t.Fatal("response leaked a connection string")
	}
}

func TestCreateDataSourceHTTPRejectsMissingActor(t *testing.T) {
	router := testRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/data-sources", bytes.NewBufferString(`{"name":"本地样例库","kind":"local_sample","connection_reference":"secret/demo"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing request id")
	}
}

func TestActorHeadersCannotEscalateWithoutAuthenticatedSession(t *testing.T) {
	router := testRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	request.Header.Set("X-Actor-ID", "admin")
	request.Header.Set("X-Actor-Role", "data_admin")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestListHTTPRejectsNonAllowlistedFiltersWithStableValidationError(t *testing.T) {
	router := testRouter(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/policies?filter_payload=secret", nil)
	request.Header.Set("Authorization", "Bearer test-admin-session")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "VALIDATION_FAILED" || body.RequestID == "" || len(body.Fields) != 1 || body.Fields[0].Field != "filter_payload" {
		t.Fatalf("body=%+v", body)
	}
}

func TestHealthReadinessCORSAndSecurityHeadersUseRuntimeMiddleware(t *testing.T) {
	router := testRouter(t)
	health := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	health.Header.Set("Origin", "http://localhost:5173")
	healthResponse := httptest.NewRecorder()
	router.ServeHTTP(healthResponse, health)
	if healthResponse.Code != http.StatusOK || !bytes.Contains(healthResponse.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Fatalf("health status=%d body=%s", healthResponse.Code, healthResponse.Body.String())
	}
	for name, expected := range map[string]string{
		"Access-Control-Allow-Origin": "http://localhost:5173",
		"X-Content-Type-Options":      "nosniff",
		"X-Frame-Options":             "DENY",
		"Referrer-Policy":             "no-referrer",
		"Content-Security-Policy":     "default-src 'self'",
	} {
		if actual := healthResponse.Header().Get(name); actual != expected {
			t.Fatalf("header %s=%q want %q", name, actual, expected)
		}
	}
	if healthResponse.Header().Get("X-Request-ID") == "" {
		t.Fatal("health response has no request id")
	}

	preflight := httptest.NewRequest(http.MethodOptions, "/api/v1/policies", nil)
	preflight.Header.Set("Origin", "http://localhost:5173")
	preflightResponse := httptest.NewRecorder()
	router.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("preflight status=%d headers=%v", preflightResponse.Code, preflightResponse.Header())
	}

	unavailable := New(nil, unavailableReadiness{err: fmt.Errorf("database unavailable")}, middleware.StaticAuthenticator{}, zap.NewNop(), time.Second)
	ready := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	readyResponse := httptest.NewRecorder()
	unavailable.ServeHTTP(readyResponse, ready)
	if readyResponse.Code != http.StatusServiceUnavailable || !bytes.Contains(readyResponse.Body.Bytes(), []byte(`"status":"not_ready"`)) || !bytes.Contains(readyResponse.Body.Bytes(), []byte(`"request_id"`)) {
		t.Fatalf("ready status=%d body=%s", readyResponse.Code, readyResponse.Body.String())
	}
}

func TestReadinessPanicIsRecoveredByRuntimeRouter(t *testing.T) {
	router := New(nil, panicReadiness{}, middleware.StaticAuthenticator{}, zap.NewNop(), time.Second)
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	request.Header.Set("X-Request-ID", "panic-request-52")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "INTERNAL_ERROR" || body["request_id"] != "panic-request-52" {
		t.Fatalf("body=%+v", body)
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("recovery response bypassed security middleware")
	}
}

func TestDiagnosisRequestDeadlineStopsReadinessIO(t *testing.T) {
	router := New(application.New(application.Dependencies{}), deadlineReadiness{}, middleware.StaticAuthenticator{}, zap.NewNop(), 500*time.Millisecond)
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	requestContext, cancel := context.WithCancel(request.Context())
	request = request.WithContext(requestContext)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(response, request)
		close(done)
	}()
	cancel()
	select {
	case <-done:
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	case <-time.After(150 * time.Millisecond):
		t.Fatal("readiness I/O ignored the request deadline")
	}
}
