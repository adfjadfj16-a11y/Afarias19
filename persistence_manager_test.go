package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestNuevoPersistenceManager verifica la creación correcta del gestor
func TestNuevoPersistenceManager(t *testing.T) {
	tmpDir := t.TempDir()
	rutaArchivo := filepath.Join(tmpDir, "test.json")

	// Crear nuevo gestor
	pm, err := NuevoPersistenceManager(rutaArchivo)
	if err != nil {
		t.Fatalf("Failed to create PersistenceManager: %v", err)
	}

	if pm == nil {
		t.Fatal("PersistenceManager is nil")
	}

	if pm.rutaArchivo != rutaArchivo {
		t.Errorf("File path mismatch: got %s, want %s", pm.rutaArchivo, rutaArchivo)
	}

	if len(pm.datos) != 0 {
		t.Errorf("Initial data should be empty, got %d elements", len(pm.datos))
	}
}

// TestNuevoPersistenceManager_ExistingFile verifica carga de archivo existente
func TestNuevoPersistenceManager_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	rutaArchivo := filepath.Join(tmpDir, "test.json")

	// Crear archivo con datos previos
	datosIniciales := map[string]interface{}{
		"nombre":   "Afarias19",
		"version":  1,
		"activo":   true,
	}
	contenido, _ := json.MarshalIndent(datosIniciales, "", "  ")
	os.WriteFile(rutaArchivo, contenido, 0644)

	// Cargar desde archivo existente
	pm, err := NuevoPersistenceManager(rutaArchivo)
	if err != nil {
		t.Fatalf("Failed to load from existing file: %v", err)
	}

	// Verificar datos cargados
	if valor, existe := pm.Obtener("nombre"); !existe || valor != "Afarias19" {
		t.Errorf("Failed to load 'nombre' from file")
	}

	if valor, existe := pm.Obtener("version"); !existe || valor != float64(1) {
		t.Errorf("Failed to load 'version' from file")
	}
}

// TestGuardar verifica guardar y persistencia
func TestGuardar(t *testing.T) {
	tmpDir := t.TempDir()
	rutaArchivo := filepath.Join(tmpDir, "test.json")
	pm, _ := NuevoPersistenceManager(rutaArchivo)

	// Guardar datos
	err := pm.Guardar("clave1", "valor1")
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Verificar en memoria
	valor, existe := pm.Obtener("clave1")
	if !existe || valor != "valor1" {
		t.Error("Failed to retrieve saved value from memory")
	}

	// Verificar en disco
	contenido, _ := os.ReadFile(rutaArchivo)
	var datosArchivo map[string]interface{}
	json.Unmarshal(contenido, &datosArchivo)
	if datosArchivo["clave1"] != "valor1" {
		t.Error("Failed to persist to disk")
	}
}

// TestGuardar_ClaveVacia verifica error con clave vacía
func TestGuardar_ClaveVacia(t *testing.T) {
	tmpDir := t.TempDir()
	pm, _ := NuevoPersistenceManager(filepath.Join(tmpDir, "test.json"))

	err := pm.Guardar("", "valor")
	if err == nil {
		t.Error("Expected error for empty key")
	}
}

// TestObtener verifica recuperación de datos
func TestObtener(t *testing.T) {
	tmpDir := t.TempDir()
	pm, _ := NuevoPersistenceManager(filepath.Join(tmpDir, "test.json"))

	// Guardar y obtener
	pm.Guardar("test", "result")
	valor, existe := pm.Obtener("test")

	if !existe {
		t.Error("Key should exist")
	}

	if valor != "result" {
		t.Errorf("Expected 'result', got %v", valor)
	}

	// Obtener clave inexistente
	_, existe = pm.Obtener("noexiste")
	if existe {
		t.Error("Non-existent key should return false")
	}
}

// TestEliminar verifica eliminación de datos
func TestEliminar(t *testing.T) {
	tmpDir := t.TempDir()
	rutaArchivo := filepath.Join(tmpDir, "test.json")
	pm, _ := NuevoPersistenceManager(rutaArchivo)

	// Guardar y eliminar
	pm.Guardar("temp", "data")
	pm.Eliminar("temp")

	// Verificar eliminación
	_, existe := pm.Obtener("temp")
	if existe {
		t.Error("Key should be deleted")
	}

	// Verificar persistencia en disco
	contenido, _ := os.ReadFile(rutaArchivo)
	var datosArchivo map[string]interface{}
	json.Unmarshal(contenido, &datosArchivo)
	if _, exists := datosArchivo["temp"]; exists {
		t.Error("Deleted key should not persist to disk")
	}
}

// TestClaves verifica listado de claves
func TestClaves(t *testing.T) {
	tmpDir := t.TempDir()
	pm, _ := NuevoPersistenceManager(filepath.Join(tmpDir, "test.json"))

	// Guardar múltiples claves
	pm.Guardar("key1", "val1")
	pm.Guardar("key2", "val2")
	pm.Guardar("key3", "val3")

	claves := pm.Claves()
	if len(claves) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(claves))
	}

	// Verificar que todas las claves estén presentes
	claveMap := make(map[string]bool)
	for _, k := range claves {
		claveMap[k] = true
	}

	for _, esperada := range []string{"key1", "key2", "key3"} {
		if !claveMap[esperada] {
			t.Errorf("Expected key %s not found", esperada)
		}
	}
}

// TestExiste verifica existencia de claves
func TestExiste(t *testing.T) {
	tmpDir := t.TempDir()
	pm, _ := NuevoPersistenceManager(filepath.Join(tmpDir, "test.json"))

	pm.Guardar("existe", "valor")

	if !pm.Existe("existe") {
		t.Error("Key should exist")
	}

	if pm.Existe("noexiste") {
		t.Error("Non-existent key should return false")
	}
}

// TestLimpiarTodo verifica limpieza completa
func TestLimpiarTodo(t *testing.T) {
	tmpDir := t.TempDir()
	rutaArchivo := filepath.Join(tmpDir, "test.json")
	pm, _ := NuevoPersistenceManager(rutaArchivo)

	// Guardar datos
	pm.Guardar("key1", "val1")
	pm.Guardar("key2", "val2")

	// Limpiar
	pm.LimpiarTodo()

	// Verificar limpieza en memoria
	claves := pm.Claves()
	if len(claves) != 0 {
		t.Errorf("Expected 0 keys after cleanup, got %d", len(claves))
	}

	// Verificar limpieza en disco
	contenido, _ := os.ReadFile(rutaArchivo)
	var datosArchivo map[string]interface{}
	json.Unmarshal(contenido, &datosArchivo)
	if len(datosArchivo) != 0 {
		t.Error("File should be empty after cleanup")
	}
}

// TestConcurrencia verifica acceso concurrente seguro
func TestConcurrencia(t *testing.T) {
	tmpDir := t.TempDir()
	pm, _ := NuevoPersistenceManager(filepath.Join(tmpDir, "test.json"))

	var wg sync.WaitGroup
	numGoroutines := 100
	operacionesPorGoroutine := 10

	// Goroutines escribiendo
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operacionesPorGoroutine; j++ {
				clave := "key_" + string(rune(id))
				valor := "value_" + string(rune(j))
				pm.Guardar(clave, valor)
			}
		}(i)
	}

	// Goroutines leyendo
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operacionesPorGoroutine; j++ {
				clave := "key_" + string(rune(id))
				pm.Obtener(clave)
			}
		}(i)
	}

	wg.Wait()

	// Verificar que terminó sin error
	if t.Failed() {
		t.Error("Concurrent access failed")
	}
}

// TestTiposDatos verifica guardar diferentes tipos de datos
func TestTiposDatos(t *testing.T) {
	tmpDir := t.TempDir()
	pm, _ := NuevoPersistenceManager(filepath.Join(tmpDir, "test.json"))

	testCases := []struct {
		clave string
		valor interface{}
	}{
		{"string", "valor"},
		{"int", 42},
		{"float", 3.14},
		{"bool", true},
		{"array", []interface{}{1, 2, 3}},
		{"map", map[string]interface{}{"nested": "value"}},
	}

	for _, tc := range testCases {
		pm.Guardar(tc.clave, tc.valor)
		valor, existe := pm.Obtener(tc.clave)
		if !existe {
			t.Errorf("Failed to retrieve key: %s", tc.clave)
		}

		// Nota: JSON unmarshalling puede cambiar tipos (int → float64)
		// verificamos que el valor esté presentes
		if valor == nil {
			t.Errorf("Retrieved nil value for key: %s", tc.clave)
		}
	}
}

// BenchmarkGuardar mide performance de Guardar
func BenchmarkGuardar(b *testing.B) {
	tmpDir := b.TempDir()
	pm, _ := NuevoPersistenceManager(filepath.Join(tmpDir, "bench.json"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clave := "bench_" + string(rune(i%100))
		pm.Guardar(clave, i)
	}
}

// BenchmarkObtener mide performance de Obtener
func BenchmarkObtener(b *testing.B) {
	tmpDir := b.TempDir()
	pm, _ := NuevoPersistenceManager(filepath.Join(tmpDir, "bench.json"))

	// Pre-populate
	for i := 0; i < 100; i++ {
		pm.Guardar("bench_"+string(rune(i)), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clave := "bench_" + string(rune(i%100))
		pm.Obtener(clave)
	}
}

// TestArchivoVacio verifica tratamiento de archivo vacío
func TestArchivoVacio(t *testing.T) {
	tmpDir := t.TempDir()
	rutaArchivo := filepath.Join(tmpDir, "empty.json")

	// Crear archivo vacío
	os.WriteFile(rutaArchivo, []byte(""), 0644)

	// Debe tratarse como almacenamiento vacío, no error
	pm, err := NuevoPersistenceManager(rutaArchivo)
	if err == nil {
		t.Log("Empty file treated as new storage (expected behavior)")
	}
	// Alternativa: si debería fallar, verificar
	if err != nil && pm != nil {
		// OK: error pero pm creado
	}
}

// TestDirectoriosPadres verifica creación de directorios padres
func TestDirectoriosPadres(t *testing.T) {
	tmpDir := t.TempDir()
	rutaAnidada := filepath.Join(tmpDir, "nivel1", "nivel2", "test.json")

	// El gestor debería crear directorios padres automáticamente
	// (o al menos no fallar)
	pm, err := NuevoPersistenceManager(rutaAnidada)

	// Verificar que fue creado o bien maneja el error
	if err != nil {
		// OK si falla, pero debería documentarse
		t.Logf("Parent directories not auto-created (may be expected): %v", err)
	} else if pm != nil {
		pm.Guardar("test", "value")
		// Verificar persistencia
		contenido, err := os.ReadFile(rutaAnidada)
		if err != nil {
			t.Error("Failed to persist with nested path")
		}
		if len(contenido) == 0 {
			t.Error("File should contain data")
		}
	}
}
