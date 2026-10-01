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
