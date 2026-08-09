package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	server, err := NewServer(Config{
		DataFile:   filepath.Join(t.TempDir(), "leads.json"),
		AdminToken: "secret-token",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}

func TestCreateLead(t *testing.T) {
	server := newTestServer(t)

	body := bytes.NewBufferString(`{"name":"Ana","email":"ana@example.com","company":"Acme","goal":"mejorar soporte"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/leads", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var lead Lead
	if err := json.Unmarshal(rec.Body.Bytes(), &lead); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if lead.Plan != "trial" {
		t.Fatalf("plan = %q, want trial", lead.Plan)
	}
	if lead.Email != "ana@example.com" {
		t.Fatalf("email = %q", lead.Email)
	}
}

func TestCreateLeadRejectsInvalidEmail(t *testing.T) {
	server := newTestServer(t)

	body := bytes.NewBufferString(`{"name":"Ana","email":"ana","company":"Acme","goal":"mejorar soporte"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/leads", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAdminLeadsRequiresToken(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/leads", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAssistant(t *testing.T) {
	server := newTestServer(t)

	body := bytes.NewBufferString(`{"message":"Necesito información del precio"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/assistant", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
