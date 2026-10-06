package errors

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Severity representa el nivel de criticidad de un error auditado.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
)

var validSeverities = map[Severity]struct{}{
	SeverityCritical: {},
	SeverityHigh:     {},
	SeverityMedium:   {},
	SeverityLow:      {},
}

// ErrorSchema representa un error auditable sin secretos.
type ErrorSchema struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Severity  Severity  `json:"severity"`
	Category  string    `json:"category"`
	Timestamp time.Time `json:"timestamp"`
	Hash256   string    `json:"hash256"`
	PrevHash  string    `json:"prev_hash,omitempty"`
}

// ErrorRegistry mantiene un índice ordenado para búsquedas O(log n).
type ErrorRegistry struct {
	mu       sync.RWMutex
	errors   []ErrorSchema
	byCode   map[string][]int
	byID     map[int64]int
	nextID   int64
	lastHash string
}

func NewRegistry() *ErrorRegistry {
	return &ErrorRegistry{
		byCode: make(map[string][]int),
		byID:   make(map[int64]int),
		nextID: 1,
	}
}

func ComputeHash(code, message string, ts time.Time, prevHash string) string {
	sum := sha256.Sum256([]byte(code + message + ts.UTC().Format(time.RFC3339Nano) + prevHash))
	return hex.EncodeToString(sum[:])
}

func (r *ErrorRegistry) Register(err ErrorSchema) (ErrorSchema, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := validSeverities[err.Severity]; !ok {
		return ErrorSchema{}, fmt.Errorf("invalid severity: %s", err.Severity)
	}
	if err.Code == "" || err.Message == "" || err.Category == "" {
		return ErrorSchema{}, fmt.Errorf("code, message and category are required")
	}
	if err.Timestamp.IsZero() {
		err.Timestamp = time.Now().UTC()
	} else {
		err.Timestamp = err.Timestamp.UTC()
	}
	if err.ID == 0 {
		err.ID = r.nextID
		r.nextID++
	}
	err.PrevHash = r.lastHash
	err.Hash256 = ComputeHash(err.Code, err.Message, err.Timestamp, err.PrevHash)

	idx := len(r.errors)
	r.errors = append(r.errors, err)
	r.byID[err.ID] = idx
	r.byCode[err.Code] = append(r.byCode[err.Code], idx)
	r.lastHash = err.Hash256
	return err, nil
}

func (r *ErrorRegistry) Lookup(code string) []ErrorSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	indexes := r.byCode[code]
	results := make([]ErrorSchema, len(indexes))
	for i, idx := range indexes {
		results[i] = r.errors[idx]
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})
	return results
}

func (r *ErrorRegistry) ValidateHash(code string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	indexes := r.byCode[code]
	for _, idx := range indexes {
		err := r.errors[idx]
		if ComputeHash(err.Code, err.Message, err.Timestamp, err.PrevHash) != err.Hash256 {
			return false
		}
	}
	return true
}

func (r *ErrorRegistry) AuditTrail() []ErrorSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	trail := make([]ErrorSchema, len(r.errors))
	copy(trail, r.errors)
	return trail
}
