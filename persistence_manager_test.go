package main

import (
	"os"
	"path/filepath"
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

