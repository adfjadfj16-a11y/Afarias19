package app

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	freeTrialDays = 30
	paidMonthly   = 19.99
	maxBodyBytes  = 1 << 20
)

//go:embed index.html
var webFS embed.FS

type Config struct {
	DataFile   string
	AdminToken string
}

type Server struct {
	mux         *http.ServeMux
	store       *LeadStore
	adminToken  string
	landingPage []byte
	now         func() time.Time
}

type Lead struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Company     string    `json:"company"`
	Goal        string    `json:"goal"`
	Plan        string    `json:"plan"`
	CreatedAt   time.Time `json:"createdAt"`
	TrialEndsAt time.Time `json:"trialEndsAt"`
}

type leadInput struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
	Goal    string `json:"goal"`
}

type assistantRequest struct {
	Message string `json:"message"`
	Context string `json:"context"`
}

type assistantResponse struct {
	Reply   string   `json:"reply"`
	Actions []string `json:"actions"`
}

type LeadStore struct {
	path  string
	mu    sync.Mutex
	leads []Lead
}

func NewServer(cfg Config) (*Server, error) {
	page, err := webFS.ReadFile("index.html")
	if err != nil {
		return nil, fmt.Errorf("read landing page: %w", err)
	}

	store, err := NewLeadStore(cfg.DataFile)
	if err != nil {
		return nil, err
	}

	s := &Server{
		mux:         http.NewServeMux(),
		store:       store,
		adminToken:  cfg.AdminToken,
		landingPage: page,
		now:         time.Now,
	}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return securityHeaders(s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleLanding)
	s.mux.HandleFunc("POST /signup", s.handleSignupForm)
	s.mux.HandleFunc("GET /api/healthz", s.handleHealth)
	s.mux.HandleFunc("GET /api/plans", s.handlePlans)
	s.mux.HandleFunc("POST /api/leads", s.handleCreateLead)
	s.mux.HandleFunc("POST /api/assistant", s.handleAssistant)
	s.mux.HandleFunc("GET /api/admin/leads", s.handleAdminLeads)
}

func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(s.landingPage)
}

func (s *Server) handleSignupForm(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}

	lead, err := s.createLead(leadInput{
		Name:    r.FormValue("name"),
		Email:   r.FormValue("email"),
		Company: r.FormValue("company"),
		Goal:    r.FormValue("goal"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(renderThankYouPage(lead.TrialEndsAt)))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePlans(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"freeTrialDays": freeTrialDays,
		"currency":      "USD",
		"monthlyPrice":  paidMonthly,
		"billingStarts": "after the first month",
		"features": []string{
			"captura de prospectos",
			"asistente de atención inicial",
			"panel administrativo protegido",
			"persistencia local segura",
		},
	})
}

func (s *Server) handleCreateLead(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	var input leadInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	lead, err := s.createLead(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, lead)
}

func (s *Server) handleAssistant(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	var input assistantRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	input.Message = strings.TrimSpace(input.Message)
	if input.Message == "" {
		http.Error(w, "el mensaje es obligatorio", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, assistantResponse{
		Reply:   buildAssistantReply(input.Message, input.Context),
		Actions: suggestActions(input.Message),
	})
}

func (s *Server) handleAdminLeads(w http.ResponseWriter, r *http.Request) {
	if s.adminToken == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Admin-Token")), []byte(s.adminToken)) != 1 {
		http.Error(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": s.store.Count(),
		"leads": s.store.List(),
	})
}

func (s *Server) createLead(input leadInput) (Lead, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.TrimSpace(input.Email)
	company := strings.TrimSpace(input.Company)
	goal := strings.TrimSpace(input.Goal)

	if name == "" {
		return Lead{}, errors.New("el nombre es obligatorio")
	}
	if company == "" {
		return Lead{}, errors.New("la empresa es obligatoria")
	}
	if goal == "" {
		return Lead{}, errors.New("el objetivo es obligatorio")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return Lead{}, errors.New("el correo no es válido")
	}

	now := s.now().UTC()
	id, err := newLeadID()
	if err != nil {
		return Lead{}, err
	}
	lead := Lead{
		ID:          id,
		Name:        name,
		Email:       email,
		Company:     company,
		Goal:        goal,
		Plan:        "trial",
		CreatedAt:   now,
		TrialEndsAt: now.Add(freeTrialDays * 24 * time.Hour),
	}

	if err := s.store.Add(lead); err != nil {
		return Lead{}, err
	}
	return lead, nil
}

func newLeadID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generar id: %w", err)
	}
	return "lead-" + hex.EncodeToString(raw[:]), nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func buildAssistantReply(message, context string) string {
	lowerMessage := strings.ToLower(message)
	lowerContext := strings.ToLower(context)

	switch {
	case strings.Contains(lowerMessage, "precio") || strings.Contains(lowerMessage, "plan"):
		return "Ofrecemos 30 días gratis para validar el servicio y luego un plan mensual simple para seguir atendiendo clientes con rapidez y seguridad."
	case strings.Contains(lowerMessage, "seguridad"):
		return "La propuesta prioriza validación de entradas, control de acceso administrativo, almacenamiento atómico y reducción de exposición de datos."
	case strings.Contains(lowerMessage, "ia") || strings.Contains(lowerMessage, "automat"):
		return "La IA del MVP ayuda a responder preguntas frecuentes y a orientar el siguiente paso sin depender todavía de proveedores externos."
	case strings.Contains(lowerContext, "ventas"):
		return "Para ventas conviene captar el prospecto, entender su objetivo y ofrecer una demo rápida durante el período gratuito."
	default:
		return "Podemos ayudarte a captar prospectos, responder preguntas frecuentes y convertir el primer mes gratis en una experiencia clara y segura."
	}
}

func suggestActions(message string) []string {
	lower := strings.ToLower(message)
	actions := []string{"registrar prospecto", "mostrar beneficios del mes gratis"}
	if strings.Contains(lower, "demo") {
		actions = append(actions, "agendar demostración")
	}
	if strings.Contains(lower, "seguridad") {
		actions = append(actions, "explicar controles de acceso")
	}
	if strings.Contains(lower, "precio") || strings.Contains(lower, "plan") {
		actions = append(actions, "explicar inicio del cobro desde el segundo mes")
	}
	return actions
}

func renderThankYouPage(trialEndsAt time.Time) string {
	const tpl = `
<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Registro confirmado</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 0; background: #0f172a; color: #e2e8f0; }
    main { max-width: 720px; margin: 4rem auto; padding: 2rem; background: #111827; border-radius: 16px; }
    a { color: #38bdf8; }
  </style>
</head>
<body>
  <main>
    <h1>Registro confirmado</h1>
    <p>Tu prueba gratuita quedó activa. El período actual termina el <strong>{{ . }}</strong>.</p>
    <p>Puedes volver a la página principal para seguir explorando la propuesta del producto.</p>
    <p><a href="/">Volver al inicio</a></p>
  </main>
</body>
</html>`

	var out bytes.Buffer
	_ = template.Must(template.New("thanks").Parse(tpl)).Execute(&out, trialEndsAt.Format("2006-01-02"))
	return out.String()
}

func NewLeadStore(path string) (*LeadStore, error) {
	store := &LeadStore{
		path:  path,
		leads: []Lead{},
	}

	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *LeadStore) Add(lead Lead) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.leads = append(s.leads, lead)
	return s.persist()
}

func (s *LeadStore) List() []Lead {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Lead, len(s.leads))
	copy(out, s.leads)
	return out
}

func (s *LeadStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.leads)
}

func (s *LeadStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read leads: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, &s.leads); err != nil {
		return fmt.Errorf("decode leads: %w", err)
	}
	return nil
}

func (s *LeadStore) persist() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	data, err := json.MarshalIndent(s.leads, "", "  ")
	if err != nil {
		return fmt.Errorf("encode leads: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp leads: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace leads file: %w", err)
	}
	return nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline' 'self'; form-action 'self'; base-uri 'self'")
		next.ServeHTTP(w, r)
	})
}
