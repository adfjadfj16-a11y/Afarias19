package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return newTestServerWithConfig(t, Config{
		DataFile:   filepath.Join(t.TempDir(), "leads.json"),
		AdminToken: "secret-token",
	})
}

func newTestServerWithConfig(t *testing.T, cfg Config) *Server {
	t.Helper()

	server, err := NewServer(cfg)
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

func TestSignupFormReturnsHTMLConfirmation(t *testing.T) {
	server := newTestServer(t)
	server.now = func() time.Time {
		return time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	}

	form := "name=Ana&email=ana%40example.com&company=Acme&goal=mejorar+soporte"
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<h1>Registro confirmado</h1>") {
		t.Fatalf("body missing confirmation heading: %q", body)
	}
	if !strings.Contains(body, "2026-02-01") {
		t.Fatalf("body missing rendered date: %q", body)
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

func TestRateLimitByIP(t *testing.T) {
	server := newTestServerWithConfig(t, Config{
		DataFile:    filepath.Join(t.TempDir(), "leads.json"),
		AdminToken:  "secret-token",
		RateLimiter: NewRateLimiter(1, time.Minute),
	})

	body1 := bytes.NewBufferString(`{"message":"precio"}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/assistant", body1)
	req1.Header.Set("Content-Type", "application/json")
	req1.RemoteAddr = "203.0.113.10:1234"
	rec1 := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", rec1.Code, http.StatusOK)
	}

	body2 := bytes.NewBufferString(`{"message":"precio"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/assistant", body2)
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "203.0.113.10:5678"
	rec2 := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", rec2.Code, http.StatusTooManyRequests)
	}
}
