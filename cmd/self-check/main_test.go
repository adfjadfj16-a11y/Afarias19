package main

import (
	"os"
	"testing"
)

// TestCheckAssets verifica que la función checkAssets detecte archivos faltantes
func TestCheckAssets(t *testing.T) {
	// Guardar directorio actual
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Cambiar a directorio temporal para evitar efectos secundarios
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// Test: Assets faltantes
	result := checkAssets()
	if result.Status != FAIL {
		t.Errorf("Expected FAIL status when assets are missing, got %s", result.Status)
	}
	if result.Remedy == "" {
		t.Error("Expected remedy message when assets are missing")
	}

	// Test: Assets presentes
	requiredAssets := []string{
		"persistence_manager.go",
		"bot_spot_binance_safe.py",
		"check-persistent-memory.js",
	}

	for _, asset := range requiredAssets {
		os.WriteFile(asset, []byte("test"), 0644)
	}

	result = checkAssets()
	if result.Status != PASS {
		t.Errorf("Expected PASS status when all assets exist, got %s", result.Status)
	}
}

// TestCheckConfig verifica que la función checkConfig valide configuración
func TestCheckConfig(t *testing.T) {
	result := checkConfig()

	// La prueba debe pasar si el sistema está configurado correctamente
	// (o puede ser FAIL si hay problemas de permisos)
	if result.Status != PASS && result.Status != FAIL && result.Status != WARN {
		t.Errorf("Expected valid status, got %s", result.Status)
	}

	// Verificar que haya un mensaje
	if result.Message == "" {
		t.Error("Expected message in check result")
	}
}

// TestCheckEnvironment verifica que la función checkEnvironment valide el entorno
func TestCheckEnvironment(t *testing.T) {
	result := checkEnvironment()

	// Debería pasar en una máquina con Go 1.24+
	if result.Status != PASS && result.Status != FAIL && result.Status != WARN {
		t.Errorf("Expected valid status, got %s", result.Status)
	}
}

// TestCheckPermissions verifica que la función checkPermissions valide permisos
func TestCheckPermissions(t *testing.T) {
	result := checkPermissions()

	// Debería verificar el binario ejecutable
	if result.Status != PASS && result.Status != FAIL && result.Status != WARN {
		t.Errorf("Expected valid status, got %s", result.Status)
	}
}

// TestRunSelfCheck verifica que la función runSelfCheck funcione correctamente
func TestRunSelfCheck(t *testing.T) {
	report := runSelfCheck()

	// Verificar que el informe tenga datos válidos
	if report.Version == "" {
		t.Error("Expected version in report")
	}

	if report.Timestamp == "" {
		t.Error("Expected timestamp in report")
	}

	if report.Status != "ok" && report.Status != "warning" && report.Status != "critical" {
		t.Errorf("Expected valid status, got %s", report.Status)
	}

	// Verificar resumen
	totalChecks := report.Summary["PASS"] + report.Summary["WARN"] + report.Summary["FAIL"]
	if totalChecks != len(report.Checks) {
		t.Errorf("Summary totals don't match checks count: %d vs %d", totalChecks, len(report.Checks))
	}

	// Verificar que haya al menos algunas verificaciones
	if len(report.Checks) == 0 {
		t.Error("Expected at least one check in report")
	}
}

// TestCheckStatusValues verifica que CheckStatus tenga valores válidos
func TestCheckStatusValues(t *testing.T) {
	tests := []struct {
		name     string
		status   CheckStatus
		expected string
	}{
		{"PASS status", PASS, "PASS"},
		{"WARN status", WARN, "WARN"},
		{"FAIL status", FAIL, "FAIL"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Status value mismatch: got %s, expected %s", string(tt.status), tt.expected)
		}
	}
}

// TestPrintReadableReport verifica que el formato legible sea generado sin errores
func TestPrintReadableReport(t *testing.T) {
	report := runSelfCheck()

	// Esta función no debería causar pánico
	// (simplemente imprime a stdout)
	// En una prueba real, podríamos capturar stdout
	printReadableReport(report, false)
}

// BenchmarkRunSelfCheck mide el tiempo de ejecución de runSelfCheck
func BenchmarkRunSelfCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runSelfCheck()
	}
}
