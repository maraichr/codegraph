package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maraichr/codegraph/pkg/apierr"
)

func TestProjectHandler_Create_InvalidBody(t *testing.T) {
	ph := &ProjectHandler{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	ph.Create(w, req)

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

func TestProjectHandler_Create_InvalidSlug(t *testing.T) {
	ph := &ProjectHandler{}
	body, _ := json.Marshal(map[string]string{
		"name": "My Project",
		"slug": "",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(body))
	w := httptest.NewRecorder()

	ph.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp apierr.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error.Code != apierr.CodeSlugRequired {
		t.Errorf("expected code %s, got %s", apierr.CodeSlugRequired, resp.Error.Code)
	}
}

func TestProjectHandler_Create_InvalidName(t *testing.T) {
	ph := &ProjectHandler{}
	body, _ := json.Marshal(map[string]string{
		"name": "",
		"slug": "valid-slug",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", bytes.NewReader(body))
	w := httptest.NewRecorder()

	ph.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp apierr.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error.Code != apierr.CodeNameRequired {
		t.Errorf("expected code %s, got %s", apierr.CodeNameRequired, resp.Error.Code)
	}
}
