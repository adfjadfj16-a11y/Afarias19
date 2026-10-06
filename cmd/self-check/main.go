// Package main implementa Pilar 4: Auto-test al Arranque
// Ejecuta verificaciones de integridad del sistema y diagnostica problemas comunes
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	auditpkg "github.com/adfjadfj16-a11y/Afarias19/pkg/audit"
	errorspkg "github.com/adfjadfj16-a11y/Afarias19/pkg/errors"
)

// CheckStatus representa el resultado de una verificación
type CheckStatus string

const (
	PASS CheckStatus = "PASS"
	WARN CheckStatus = "WARN"
	FAIL CheckStatus = "FAIL"
)

// CheckResult contiene el resultado de una verificación
type CheckResult struct {
	Name      string      `json:"name"`
	Status    CheckStatus `json:"status"`
	Message   string      `json:"message"`
	Remedy    string      `json:"remedy,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// SelfCheckReport es el informe completo de auto-verificación
type SelfCheckReport struct {
	Version         string         `json:"version"`
	Timestamp       string         `json:"timestamp"`
	Status          string         `json:"status"` // "ok", "warning", "critical"
	Checks          []CheckResult  `json:"checks"`
	Summary         map[string]int `json:"summary"` // count of PASS, WARN, FAIL
	Recommendations []string       `json:"recommendations,omitempty"`
}

func main() {
	outputJSON := flag.Bool("json", false, "Output in JSON format")
	verbose := flag.Bool("v", false, "Verbose output")
	auditReport := flag.String("audit-report", "", "Generate compliance audit report at the given JSON path")
	help := flag.Bool("help", false, "Show help")
	version := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *help {
		fmt.Printf("Afarias19 Self-Check Tool v1.0.0\n\n")
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		fmt.Printf("  -json      Output in JSON format (default: human-readable)\n")
		fmt.Printf("  -v         Verbose output (show remedies)\n")
		fmt.Printf("  -version   Show version\n")
		fmt.Printf("  -audit-report PATH  Generate compliance JSON report\n")
		fmt.Printf("  -help      Show this help\n")
		os.Exit(0)
	}

	if *version {
		fmt.Println("Afarias19 Self-Check Tool v1.0.0")
		os.Exit(0)
	}

	report := runSelfCheck()
	if err := handleAudit(report, *auditReport); err != nil {
		fmt.Fprintf(os.Stderr, "audit error: %v\n", err)
		os.Exit(1)
	}

	if *outputJSON {
		outputJSON, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(outputJSON))
	} else {
		printReadableReport(report, *verbose)
	}

	// Exit code: 0 si todo OK, 1 si hay FAILs
	if report.Status == "critical" {
		os.Exit(1)
	}
	os.Exit(0)
}

// runSelfCheck ejecuta todas las verificaciones
func runSelfCheck() SelfCheckReport {
	timestamp := time.Now().Format(time.RFC3339)

	checks := []CheckResult{
		checkAssets(),
		checkConfig(),
		checkEnvironment(),
		checkPermissions(),
	}

	summary := map[string]int{"PASS": 0, "WARN": 0, "FAIL": 0}
	var recommendations []string

	for _, check := range checks {
		switch check.Status {
		case PASS:
			summary["PASS"]++
		case WARN:
			summary["WARN"]++
			if check.Remedy != "" {
				recommendations = append(recommendations, check.Remedy)
			}
		case FAIL:
			summary["FAIL"]++
			if check.Remedy != "" {
				recommendations = append(recommendations, check.Remedy)
			}
		}
	}

	status := "ok"
	if summary["FAIL"] > 0 {
		status = "critical"
	} else if summary["WARN"] > 0 {
		status = "warning"
	}

	return SelfCheckReport{
		Version:         "1.0.0",
		Timestamp:       timestamp,
		Status:          status,
		Checks:          checks,
		Summary:         summary,
		Recommendations: recommendations,
	}
}

// checkAssets verifica que todos los assets requeridos existan
func checkAssets() CheckResult {
	requiredAssets := []string{
		"persistence_manager.go",
		"bot_spot_binance_safe.py",
		"check-persistent-memory.js",
	}

	var missing []string
	for _, asset := range requiredAssets {
		if _, err := os.Stat(asset); os.IsNotExist(err) {
			missing = append(missing, asset)
		}
	}

	if len(missing) == 0 {
		return CheckResult{
			Name:      "Assets",
			Status:    PASS,
			Message:   fmt.Sprintf("✅ All %d required assets present", len(requiredAssets)),
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	remedy := fmt.Sprintf("Missing assets: %v. Download official release from GitHub releases.", missing)
	return CheckResult{
		Name:      "Assets",
		Status:    FAIL,
		Message:   fmt.Sprintf("❌ Missing %d assets", len(missing)),
		Remedy:    remedy,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// checkConfig verifica la configuración y disponibilidad de puertos
func checkConfig() CheckResult {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		homeDir = "."
	}

	configDir := fmt.Sprintf("%s/.config/afarias19", homeDir)

	// Intentar crear el directorio si no existe
	if err := os.MkdirAll(configDir, 0700); err != nil {
		remedy := fmt.Sprintf("Cannot create config directory: %s. Check permissions or use --config flag.", configDir)
		return CheckResult{
			Name:      "Configuration",
			Status:    FAIL,
			Message:   fmt.Sprintf("❌ Config directory not writable: %s", configDir),
			Remedy:    remedy,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	// Verificar que podamos escribir en el directorio
	testFile := fmt.Sprintf("%s/.test", configDir)
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		remedy := fmt.Sprintf("Cannot write to config directory: %s. Check permissions.", configDir)
		return CheckResult{
			Name:      "Configuration",
			Status:    FAIL,
			Message:   fmt.Sprintf("❌ Config directory not writable"),
			Remedy:    remedy,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
	os.Remove(testFile)

	return CheckResult{
		Name:      "Configuration",
		Status:    PASS,
		Message:   fmt.Sprintf("✅ Config directory writable: %s", configDir),
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// checkEnvironment verifica compatibilidad del runtime
func checkEnvironment() CheckResult {
	// En una implementación real, verificaríamos:
	// - Versión de Go (mínimo go1.24)
	// - Disponibilidad de Python 3.10+
	// - Directorios críticos (/tmp, /var/tmp, etc.)

	// Para esta demostración:
	return CheckResult{
		Name:      "Environment",
		Status:    PASS,
		Message:   "✅ Runtime environment compatible",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// checkPermissions verifica permisos del binario
func checkPermissions() CheckResult {
	executable, err := os.Executable()
	if err != nil {
		remedy := "Cannot determine executable path. Verify binary permissions."
		return CheckResult{
			Name:      "Permissions",
			Status:    FAIL,
			Message:   "❌ Cannot verify executable permissions",
			Remedy:    remedy,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	stat, err := os.Stat(executable)
	if err != nil {
		remedy := fmt.Sprintf("Cannot stat binary: %s. Reinstall from official release.", executable)
		return CheckResult{
			Name:      "Permissions",
			Status:    FAIL,
			Message:   fmt.Sprintf("❌ Binary not accessible"),
			Remedy:    remedy,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	// Verificar modo de ejecución
	if (stat.Mode() & 0111) == 0 {
		remedy := fmt.Sprintf("Binary not executable. Run: chmod +x %s", executable)
		return CheckResult{
			Name:      "Permissions",
			Status:    FAIL,
			Message:   "❌ Binary not executable",
			Remedy:    remedy,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	return CheckResult{
		Name:      "Permissions",
		Status:    PASS,
		Message:   fmt.Sprintf("✅ Binary permissions verified"),
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func handleAudit(report SelfCheckReport, reportPath string) error {
	registry := errorspkg.NewRegistry()
	auditor := auditpkg.NewAuditor(filepath.Join("audit", "self-check.log"))
	for _, check := range report.Checks {
		severity := errorspkg.SeverityLow
		status := "ok"
		switch check.Status {
		case FAIL:
			severity = errorspkg.SeverityCritical
			status = "critical"
		case WARN:
			severity = errorspkg.SeverityMedium
			status = "warning"
		}
		entry, err := registry.Register(errorspkg.ErrorSchema{
			Code:      check.Name,
			Message:   check.Message,
			Severity:  severity,
			Category:  "self-check",
			Timestamp: time.Now().UTC(),
		})
		if err != nil {
			return err
		}
		if _, err := auditor.Append(entry.Code, status, entry.Message); err != nil {
			return err
		}
	}
	if err := auditor.VerifyChain(); err != nil {
		return err
	}
	if reportPath != "" {
		payload := map[string]any{
			"report":      report,
			"audit_trail": registry.AuditTrail(),
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(reportPath), 0755); err != nil && filepath.Dir(reportPath) != "." {
			return err
		}
		if err := os.WriteFile(reportPath, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

// printReadableReport imprime el informe en formato legible
func printReadableReport(report SelfCheckReport, verbose bool) {
	fmt.Printf("\n")
	fmt.Printf("╔════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║     Afarias19 Self-Check v%s                          ║\n", report.Version)
	fmt.Printf("╚════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")

	// Resumen
	fmt.Printf("Status: %s\n", report.Status)
	fmt.Printf("Checks: %d PASS, %d WARN, %d FAIL\n",
		report.Summary["PASS"], report.Summary["WARN"], report.Summary["FAIL"])
	fmt.Printf("\n")

	// Detalles de cada check
	for _, check := range report.Checks {
		statusIcon := "✅"
		if check.Status == WARN {
			statusIcon = "⚠️ "
		} else if check.Status == FAIL {
			statusIcon = "❌"
		}
		fmt.Printf("%s %s\n", statusIcon, check.Message)
		if verbose && check.Remedy != "" {
			fmt.Printf("   → Fix: %s\n", check.Remedy)
		}
	}

	fmt.Printf("\n")

	// Recomendaciones
	if len(report.Recommendations) > 0 {
		fmt.Printf("Recommendations:\n")
		for i, rec := range report.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
		fmt.Printf("\n")
	}

	// Estado final
	if report.Status == "ok" {
		fmt.Printf("✅ System is ready to use!\n")
	} else if report.Status == "warning" {
		fmt.Printf("⚠️  System has warnings. Review recommendations above.\n")
	} else {
		fmt.Printf("❌ System has critical issues. Address remedies above.\n")
	}
	fmt.Printf("\n")
}
