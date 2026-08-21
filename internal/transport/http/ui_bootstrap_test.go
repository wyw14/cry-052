package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/wyw14/cry052/internal/middleware"
	"go.uber.org/zap"
)

type readyUI struct{}

func (readyUI) Ping(context.Context) error { return nil }

func TestVueBootstrapPublishesCompleteGovernanceNavigation(t *testing.T) {
	auth := middleware.StaticAuthenticator{"ui-session": {ActorID: "admin", Role: "data_admin"}}
	router := New(nil, readyUI{}, auth, zap.NewNop(), time.Second)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ui/bootstrap", nil)
	request.Header.Set("Authorization", "Bearer ui-session")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("bootstrap status=%d body=%s", response.Code, response.Body.String())
	}
	var got UIBootstrap
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode bootstrap: %v", err)
	}
	wantRoutes := []string{"/data-sources", "/catalog", "/policies", "/previews", "/batches", "/reports", "/audit", "/versions"}
	if got.Framework != "vue3" || got.Locale != "zh-CN" || got.Version != 3 || !reflect.DeepEqual(got.Routes, wantRoutes) {
		t.Fatalf("incomplete Vue governance bootstrap: %#v", got)
	}
}
