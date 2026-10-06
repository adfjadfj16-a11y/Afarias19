package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditLog representa un bloque auditable inmutable.
type AuditLog struct {
	Timestamp       time.Time `json:"timestamp"`
	ErrorCode       string    `json:"error_code"`
	Hash256Previous string    `json:"hash256_previous"`
	Hash256Current  string    `json:"hash256_current"`
	Status          string    `json:"status"`
	Message         string    `json:"message,omitempty"`
}

// Auditor persiste una cadena de hashes en archivo JSONL.
type Auditor struct {
	mu       sync.Mutex
	filePath string
}

func NewAuditor(filePath string) *Auditor {
	return &Auditor{filePath: filePath}
}

func ComputeCurrentHash(ts time.Time, code, status, message, prev string) string {
	sum := sha256.Sum256([]byte(ts.UTC().Format(time.RFC3339Nano) + code + status + message + prev))
	return hex.EncodeToString(sum[:])
}

func (a *Auditor) Append(errorCode, status, message string) (AuditLog, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	logs, err := a.loadLogs()
	if err != nil {
		return AuditLog{}, err
	}
	prev := ""
	if len(logs) > 0 {
		prev = logs[len(logs)-1].Hash256Current
	}
	entry := AuditLog{
		Timestamp:       time.Now().UTC(),
		ErrorCode:       errorCode,
		Hash256Previous: prev,
		Status:          status,
		Message:         message,
	}
	entry.Hash256Current = ComputeCurrentHash(entry.Timestamp, entry.ErrorCode, entry.Status, entry.Message, entry.Hash256Previous)
	if err := os.MkdirAll(filepath.Dir(a.filePath), 0755); err != nil {
		return AuditLog{}, err
	}
	f, err := os.OpenFile(a.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return AuditLog{}, err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(entry); err != nil {
		return AuditLog{}, err
	}
	return entry, nil
}

func (a *Auditor) VerifyChain() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	logs, err := a.loadLogs()
	if err != nil {
		return err
	}
	prev := ""
	for i, entry := range logs {
		if entry.Hash256Previous != prev {
			return fmt.Errorf("broken chain at index %d", i)
		}
		expected := ComputeCurrentHash(entry.Timestamp, entry.ErrorCode, entry.Status, entry.Message, entry.Hash256Previous)
		if expected != entry.Hash256Current {
			return fmt.Errorf("invalid hash at index %d", i)
		}
		prev = entry.Hash256Current
	}
	return nil
}

func (a *Auditor) Load() ([]AuditLog, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.loadLogs()
}

func (a *Auditor) loadLogs() ([]AuditLog, error) {
	f, err := os.Open(a.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditLog{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var logs []AuditLog
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry AuditLog
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		logs = append(logs, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}
