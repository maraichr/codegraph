package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maraichr/codegraph/pkg/apierr"
)

func TestSourceHandler_Create_InvalidBody(t *testing.T) {
	sh := &SourceHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/slug/sources", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	sh.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp apierr.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error.Code != apierr.CodeInvalidRequestBody {
		t.Errorf("expected code %s, got %s", apierr.CodeInvalidRequestBody, resp.Error.Code)
	}
}

func TestSourceHandler_Get_InvalidID(t *testing.T) {
	sh := &SourceHandler{}
	// sourceID will be empty string from chi params — not a valid UUID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/slug/sources/not-a-uuid", nil)
	w := httptest.NewRecorder()

	sh.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp apierr.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error.Code != apierr.CodeInvalidSourceID {
		t.Errorf("expected code %s, got %s", apierr.CodeInvalidSourceID, resp.Error.Code)
	}
}
