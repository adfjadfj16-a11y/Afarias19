package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNuevoPersistenceManager_CreaDirectorioYArchivoVacio(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "data")
	path := filepath.Join(dir, "datos.json")

	pm, err := NuevoPersistenceManager(path)
	if err != nil {
		t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
	}
	if pm == nil {
		t.Fatal("NuevoPersistenceManager devolvió nil")
	}
	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("No se debería devolver error al comprobar el archivo: %v", err)
	}
	if !pm.Existe("noexiste") {
		if len(pm.Claves()) != 0 {
			t.Fatalf("El mapa debe mantenerse vacío, pero tiene claves: %v", pm.Claves())
		}
	}
}

func TestGuardarYObtenerConUnArchivoVacio(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	path := filepath.Join(dir, "datos.json")

	pm, err := NuevoPersistenceManager(path)
	if err != nil {
		t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
	}

	if err := pm.Guardar("nombre", "Afarias19"); err != nil {
		t.Fatalf("Guardar falló: %v", err)
	}

	valor, ok := pm.Obtener("nombre")
	if !ok {
		t.Fatal("No se encontró la clave guardada")
	}
	if valor != "Afarias19" {
		t.Fatalf("Valor incorrecto: got %v want Afarias19", valor)
	}
}

func TestGuardarMultiplesValoresYObtener(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	path := filepath.Join(dir, "datos.json")

	pm, err := NuevoPersistenceManager(path)
	if err != nil {
		t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
	}

	// Guardar múltiples valores de diferentes tipos
	tests := []struct {
		clave string
		valor interface{}
	}{
		{"nombre", "Afarias19"},
		{"version", float64(1)},
		{"activo", true},
		{"contador", float64(42)},
	}

	for _, test := range tests {
		if err := pm.Guardar(test.clave, test.valor); err != nil {
			t.Fatalf("Guardar(%q, %v) falló: %v", test.clave, test.valor, err)
		}
	}

	// Verificar que todos los valores se recuperan correctamente
	for _, test := range tests {
		valor, ok := pm.Obtener(test.clave)
		if !ok {
			t.Fatalf("Clave %q no encontrada", test.clave)
		}
		if valor != test.valor {
			t.Fatalf("Valor incorrecto para clave %q: got %v want %v", test.clave, valor, test.valor)
		}
	}

	// Verificar claves
	claves := pm.Claves()
	if len(claves) != len(tests) {
		t.Fatalf("Número incorrecto de claves: got %d want %d", len(claves), len(tests))
	}
}

func TestEliminarClave(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	path := filepath.Join(dir, "datos.json")

	pm, err := NuevoPersistenceManager(path)
	if err != nil {
		t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
	}

	// Guardar una clave
	if err := pm.Guardar("temp", "data"); err != nil {
		t.Fatalf("Guardar falló: %v", err)
	}

	// Verificar que existe
	if !pm.Existe("temp") {
		t.Fatal("Clave 'temp' debería existir después de guardar")
	}

	// Eliminar la clave
	if err := pm.Eliminar("temp"); err != nil {
		t.Fatalf("Eliminar falló: %v", err)
	}

	// Verificar que ya no existe
	if pm.Existe("temp") {
		t.Fatal("Clave 'temp' no debería existir después de eliminar")
	}

	_, ok := pm.Obtener("temp")
	if ok {
		t.Fatal("Obtener('temp') no debería encontrar nada después de eliminar")
	}
}

func TestPersistenciaEntreSesiones(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	path := filepath.Join(dir, "datos.json")

	// Primera sesión: guardar datos
	{
		pm, err := NuevoPersistenceManager(path)
		if err != nil {
			t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
		}

		if err := pm.Guardar("persistido", "valor importante"); err != nil {
			t.Fatalf("Guardar falló: %v", err)
		}
	}

	// Segunda sesión: verificar que los datos se cargaron
	{
		pm, err := NuevoPersistenceManager(path)
		if err != nil {
			t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
		}

		valor, ok := pm.Obtener("persistido")
		if !ok {
			t.Fatal("Datos no fueron persistidos entre sesiones")
		}
		if valor != "valor importante" {
			t.Fatalf("Valor incorrecto después de recargar: got %v want 'valor importante'", valor)
		}
	}
}

func TestLimpiarTodo(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	path := filepath.Join(dir, "datos.json")

	pm, err := NuevoPersistenceManager(path)
	if err != nil {
		t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
	}

	// Guardar varios valores
	if err := pm.Guardar("key1", "value1"); err != nil {
		t.Fatalf("Guardar falló: %v", err)
	}
	if err := pm.Guardar("key2", "value2"); err != nil {
		t.Fatalf("Guardar falló: %v", err)
	}

	// Limpiar todo
	if err := pm.LimpiarTodo(); err != nil {
		t.Fatalf("LimpiarTodo falló: %v", err)
	}

	// Verificar que no hay claves
	if len(pm.Claves()) != 0 {
		t.Fatalf("El almacenamiento debería estar vacío después de LimpiarTodo, pero tiene: %v", pm.Claves())
	}

	// Verificar que Existe devuelve false
	if pm.Existe("key1") || pm.Existe("key2") {
		t.Fatal("Las claves no deberían existir después de LimpiarTodo")
	}
}

func TestGuardarClavesVacias(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	path := filepath.Join(dir, "datos.json")

	pm, err := NuevoPersistenceManager(path)
	if err != nil {
		t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
	}

	// Intentar guardar una clave vacía debe fallar
	if err := pm.Guardar("", "valor"); err == nil {
		t.Fatal("Guardar con clave vacía debería devolver un error")
	}
}

// TestConcurrentWrites prueba la seguridad de la concurrencia con 50 goroutines
// escribiendo diferentes claves simultáneamente.
func TestConcurrentWrites(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	path := filepath.Join(dir, "datos.json")

	pm, err := NuevoPersistenceManager(path)
	if err != nil {
		t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
	}

	// 50 goroutines escribiendo diferentes claves simultáneamente
	numGoroutines := 50
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			clave := fmt.Sprintf("key_%d", index)
			valor := fmt.Sprintf("value_%d", index)
			if err := pm.Guardar(clave, valor); err != nil {
				errChan <- fmt.Errorf("Guardar(%q, %q) falló: %w", clave, valor, err)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verificar que no hubo errores en las escrituras
	for err := range errChan {
		if err != nil {
			t.Fatalf("Error en goroutine: %v", err)
		}
	}

	// Verificar que todas las claves se escribieron correctamente
	claves := pm.Claves()
	if len(claves) != numGoroutines {
		t.Fatalf("Número incorrecto de claves: got %d want %d", len(claves), numGoroutines)
	}

	// Verificar que cada clave tiene el valor correcto
	for i := 0; i < numGoroutines; i++ {
		clave := fmt.Sprintf("key_%d", i)
		valorEsperado := fmt.Sprintf("value_%d", i)
		valor, ok := pm.Obtener(clave)
		if !ok {
			t.Fatalf("Clave %q no encontrada después de escrituras concurrentes", clave)
		}
		if valor != valorEsperado {
			t.Fatalf("Valor incorrecto para clave %q: got %v want %v", clave, valor, valorEsperado)
		}
	}
}

// TestConcurrentReadsAndWrites prueba la seguridad de la concurrencia con múltiples
// goroutines leyendo y escribiendo simultáneamente.
func TestConcurrentReadsAndWrites(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	path := filepath.Join(dir, "datos.json")

	pm, err := NuevoPersistenceManager(path)
	if err != nil {
		t.Fatalf("NuevoPersistenceManager devolvió error inesperado: %v", err)
	}

	// Guardar algunos valores iniciales
	for i := 0; i < 10; i++ {
		clave := fmt.Sprintf("initial_key_%d", i)
		valor := fmt.Sprintf("initial_value_%d", i)
		if err := pm.Guardar(clave, valor); err != nil {
			t.Fatalf("Guardar inicial falló: %v", err)
		}
	}

	numReaders := 25
	numWriters := 25

	var wg sync.WaitGroup

	// 25 goroutines leyendo
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// Leer todas las claves múltiples veces
			for j := 0; j < 100; j++ {
				pm.Claves()
				for k := 0; k < 10; k++ {
					clave := fmt.Sprintf("initial_key_%d", k)
					pm.Obtener(clave)
					pm.Existe(clave)
				}
			}
		}(i)
	}

	// 25 goroutines escribiendo
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			// Escribir nuevas claves
			for j := 0; j < 10; j++ {
				clave := fmt.Sprintf("concurrent_key_%d_%d", index, j)
				valor := fmt.Sprintf("concurrent_value_%d_%d", index, j)
				pm.Guardar(clave, valor)
			}
		}(i)
	}

	wg.Wait()

	// Verificar que el estado final es consistente
	claves := pm.Claves()
	// Deberíamos tener las 10 claves iniciales + (25 writers * 10 nuevas claves)
	expectedMinKeys := 10 + (25 * 10)
	if len(claves) < expectedMinKeys {
		t.Fatalf("Número incorrecto de claves: got %d, expected at least %d", len(claves), expectedMinKeys)
	}

	// Verificar que todas las claves iniciales aún existen
	for i := 0; i < 10; i++ {
		clave := fmt.Sprintf("initial_key_%d", i)
		if !pm.Existe(clave) {
			t.Fatalf("Clave inicial %q no encontrada", clave)
		}
	}
}

