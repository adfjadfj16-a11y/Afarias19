package errors

import (
	"fmt"
	"testing"
	"time"
)

func TestRegisterLookupAndValidate10K(t *testing.T) {
	registry := NewRegistry()
	base := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 10000; i++ {
		_, err := registry.Register(ErrorSchema{
			Code:      fmt.Sprintf("ERR-%05d", i%100),
			Message:   fmt.Sprintf("public message %d", i),
			Severity:  SeverityHigh,
			Category:  "compliance",
			Timestamp: base.Add(time.Duration(i) * time.Millisecond),
		})
		if err != nil {
			t.Fatalf("register failed: %v", err)
		}
	}

	matches := registry.Lookup("ERR-00042")
	if len(matches) != 100 {
		t.Fatalf("expected 100 matches, got %d", len(matches))
	}
	if !registry.ValidateHash("ERR-00042") {
		t.Fatal("expected valid hash chain for code")
	}
	if len(registry.AuditTrail()) != 10000 {
		t.Fatal("expected full audit trail")
	}
}

func TestRegisterRejectsInvalidSeverity(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.Register(ErrorSchema{Code: "E", Message: "m", Category: "c", Severity: Severity("BAD")})
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
}
