package httpapi

import (
	"errors"
	"net/http"
	"testing"

	"github.com/wyw14/cry052/internal/application"
	"github.com/wyw14/cry052/internal/domain"
)

func TestAPIErrorContractPreservesCauseFieldsAndRequestIdentity(t *testing.T) {
	_, cause := application.ValidateDataSourcePage(domain.PageRequest{Page: 1, Size: 101, Sort: "created_at"})
	err := errors.Join(errors.New("list data sources"), cause)
	contract := buildErrorContract(err, "request-24")
	if contract.Status != http.StatusBadRequest || contract.Code != "VALIDATION_FAILED" || contract.Message != "invalid list query" {
		t.Fatalf("wrong public error contract: %#v", contract)
	}
	if contract.RequestID != "request-24" || len(contract.Fields) != 1 || contract.Fields[0].Field != "size" {
		t.Fatalf("missing field or request identity: %#v", contract)
	}
	if err := validatePageRequest(domain.PageRequest{Page: 1, Size: 101, Sort: "created_at"}); err == nil {
		t.Fatal("oversized page was accepted")
	}
}
