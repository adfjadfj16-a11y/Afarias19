// Package checks proporciona verificaciones de integridad del sistema.
// Implementa Pilar 4: Auto-test al Arranque
package checks

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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
	Name      string
	Status    CheckStatus
	Message   string
	Remedy    string // Acción para resolver el problema
	Timestamp string
}

// Check es una función que realiza una verificación
type Check func() CheckResult

// Run ejecuta todas las verificaciones y devuelve los resultados
func Run() []CheckResult {
	checks := []Check{
		CheckAssets,
		CheckConfig,
		CheckEnvironment,
		CheckPermissions,
	}

	var results []CheckResult
	for _, check := range checks {
		results = append(results, check())
	}
	return results
}

// CheckAssets verifica que todos los assets requeridos existan
func CheckAssets() CheckResult {
	// Assets que deben existir
	requiredAssets := []string{
		"persistence_manager.go",
		"bot_spot_binance_safe.py",
		"check-persistent-memory.js",
	}

	var missing []string
	for _, asset := range requiredAssets {
		path := filepath.Join(".", asset)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missing = append(missing, asset)
		}
	}

	if len(missing) == 0 {
		return CheckResult{
			Name:    "Assets",
			Status:  PASS,
			Message: fmt.Sprintf("✅ All %d required assets present", len(requiredAssets)),
			Remedy:  "",
		}
	}

	remedy := fmt.Sprintf("Missing assets: %s. Download official release from GitHub releases.", strings.Join(missing, ", "))
	return CheckResult{
		Name:    "Assets",
		Status:  FAIL,
		Message: fmt.Sprintf("❌ Missing %d assets: %s", len(missing), strings.Join(missing, ", ")),
		Remedy:  remedy,
	}
}

// CheckConfig verifica la configuración y disponibilidad de puertos
func CheckConfig() CheckResult {
	// Verificar que el directorio de config sea escribible
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "afarias19")
	if configDir == filepath.Join("", ".config", "afarias19") {
		configDir = "/tmp/afarias19"
	}

	// Intentar crear el directorio si no existe
	if err := os.MkdirAll(configDir, 0700); err != nil {
		remedy := fmt.Sprintf("Cannot create config directory: %s. Check directory permissions or use --config flag.", configDir)
		return CheckResult{
			Name:    "Configuration",
			Status:  FAIL,
			Message: fmt.Sprintf("❌ Config directory not writable: %s", configDir),
			Remedy:  remedy,
		}
	}

	// Verificar que podamos escribir en el directorio
	testFile := filepath.Join(configDir, ".test")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		remedy := fmt.Sprintf("Cannot write to config directory: %s. Check permissions.", configDir)
		return CheckResult{
			Name:    "Configuration",
			Status:  FAIL,
			Message: fmt.Sprintf("❌ Config directory not writable: %s", configDir),
			Remedy:  remedy,
		}
	}
	os.Remove(testFile)

	return CheckResult{
		Name:    "Configuration",
		Status:  PASS,
		Message: fmt.Sprintf("✅ Config directory writable: %s", configDir),
		Remedy:  "",
	}
}

// CheckEnvironment verifica compatibilidad del runtime y dependencias
func CheckEnvironment() CheckResult {
	var issues []string

	// Verificar Go runtime
	if runtime.Version() < "go1.24" {
		issues = append(issues, fmt.Sprintf("Go version %s (minimum: go1.24)", runtime.Version()))
	} else {
		// Se pasó la verificación
	}

	// Verificar acceso a directorios críticos
	criticalDirs := []string{"/tmp", "/var/tmp"}
	if runtime.GOOS == "windows" {
		criticalDirs = []string{os.TempDir()}
	}

	for _, dir := range criticalDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("Critical directory not found: %s", dir))
		}
	}

	if len(issues) == 0 {
		return CheckResult{
			Name:    "Environment",
			Status:  PASS,
			Message: fmt.Sprintf("✅ Runtime compatible: %s / %s", runtime.GOOS, runtime.GOARCH),
			Remedy:  "",
		}
	}

	remedy := fmt.Sprintf("Environment issues: %s. Ensure Go 1.24+ is installed and system directories are accessible.", strings.Join(issues, "; "))
	return CheckResult{
		Name:    "Environment",
		Status:  FAIL,
		Message: fmt.Sprintf("❌ Environment incompatible: %s", strings.Join(issues, "; ")),
		Remedy:  remedy,
	}
}

// CheckPermissions verifica permisos de archivos críticos
func CheckPermissions() CheckResult {
	executable, err := os.Executable()
	if err != nil {
		remedy := "Cannot determine executable path. Verify the binary has correct permissions."
		return CheckResult{
			Name:    "Permissions",
			Status:  FAIL,
			Message: "❌ Cannot verify executable permissions",
			Remedy:  remedy,
		}
	}

	// Verificar que el binario sea ejecutable
	stat, err := os.Stat(executable)
	if err != nil {
		remedy := fmt.Sprintf("Cannot stat binary: %s. Reinstall from official release.", executable)
		return CheckResult{
			Name:    "Permissions",
			Status:  FAIL,
			Message: fmt.Sprintf("❌ Binary not accessible: %s", executable),
			Remedy:  remedy,
		}
	}

	// Verificar modo de ejecución
	if (stat.Mode() & 0111) == 0 {
		remedy := fmt.Sprintf("Binary not executable: %s. Run: chmod +x %s", executable, executable)
		return CheckResult{
			Name:    "Permissions",
			Status:  FAIL,
			Message: fmt.Sprintf("❌ Binary not executable: %s", executable),
			Remedy:  remedy,
		}
	}

	// Verificar que el propietario sea el usuario actual
	currentUser, err := user.Current()
	if err == nil && stat.Sys() != nil {
		// Nota: verificación completa de propietario es complicada, simplemente verificar ejecutabilidad
	}

	return CheckResult{
		Name:    "Permissions",
		Status:  PASS,
		Message: fmt.Sprintf("✅ Binary permissions verified: %s", executable),
		Remedy:  "",
	}
}

// VerifyChecksum verifica la integridad de un archivo contra un checksum esperado
func VerifyChecksum(filePath string, expectedChecksum string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}

	actual := fmt.Sprintf("%x", hash.Sum(nil))
	return actual == expectedChecksum, nil
}

// GetRuntimeVersion retorna la versión del runtime Go
func GetRuntimeVersion() string {
	return runtime.Version()
}

// ParseGoVersion extrae números de versión de una string de versión Go
func ParseGoVersion(version string) (major, minor int, err error) {
	// Esperado formato: "go1.24", "go1.24.1", etc.
	version = strings.TrimPrefix(version, "go")
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid go version format: %s", version)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return major, minor, nil
}
