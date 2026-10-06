package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendAndVerifyChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	auditor := NewAuditor(path)
	for i := 0; i < 3; i++ {
		if _, err := auditor.Append("ERR-CRITICAL", "critical", "public message"); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}
	if err := auditor.VerifyChain(); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerifyChainDetectsTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	auditor := NewAuditor(path)
	if _, err := auditor.Append("ERR-1", "critical", "one"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if _, err := auditor.Append("ERR-2", "critical", "two"); err != nil {
		t.Fatalf("append failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	tampered := strings.Replace(string(data), "two", "evil", 1)
	if err := os.WriteFile(path, []byte(tampered), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := auditor.VerifyChain(); err == nil {
		t.Fatal("expected tampering detection")
	}
}
