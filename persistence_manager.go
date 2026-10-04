// Package main es el punto de entrada del programa.
// Aquí definimos todo lo relacionado con guardar y cargar datos en un archivo.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// PersistenceManager guarda datos en un archivo JSON local y privado.
type PersistenceManager struct {
	rutaArchivo string
	datos       map[string]interface{}
	mu          sync.RWMutex
}

// NuevoPersistenceManager crea un gestor con persistencia local privada.
func NuevoPersistenceManager(rutaArchivo string) (*PersistenceManager, error) {
	if rutaArchivo == "" {
		return nil, errors.New("la ruta del archivo no puede estar vacía")
	}

	if err := os.MkdirAll(filepath.Dir(rutaArchivo), 0o755); err != nil {
		return nil, fmt.Errorf("no se pudo crear el directorio padre: %w", err)
	}

	pm := &PersistenceManager{
		rutaArchivo: rutaArchivo,
		datos:       make(map[string]interface{}),
	}

	if err := pm.cargar(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return pm, nil
}

func (pm *PersistenceManager) Guardar(clave string, valor interface{}) error {
	if clave == "" {
		return errors.New("la clave no puede estar vacía")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.datos[clave] = valor
	return pm.persistir()
}

func (pm *PersistenceManager) Obtener(clave string) (interface{}, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	valor, existe := pm.datos[clave]
	return valor, existe
}

func (pm *PersistenceManager) Eliminar(clave string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.datos, clave)
	return pm.persistir()
}

func (pm *PersistenceManager) Claves() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	claves := make([]string, 0, len(pm.datos))
	for clave := range pm.datos {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	return claves
}

func (pm *PersistenceManager) Existe(clave string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, existe := pm.datos[clave]
	return existe
}

func (pm *PersistenceManager) LimpiarTodo() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.datos = make(map[string]interface{})
	return pm.persistir()
}

func (pm *PersistenceManager) persistir() error {
	contenido, err := json.MarshalIndent(pm.datos, "", "  ")
	if err != nil {
		return err
	}

	rutaTemporal := pm.rutaArchivo + ".tmp"
	if err := os.WriteFile(rutaTemporal, contenido, 0o600); err != nil {
		return err
	}

	if err := os.Chmod(pm.rutaArchivo, 0o600); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(rutaTemporal, pm.rutaArchivo); err != nil {
		return err
	}

	return os.Chmod(pm.rutaArchivo, 0o600)
}

func (pm *PersistenceManager) cargar() error {
	contenido, err := os.ReadFile(pm.rutaArchivo)
	if err != nil {
		return err
	}

	if len(contenido) == 0 {
		pm.datos = make(map[string]interface{})
		return nil
	}

	if err := json.Unmarshal(contenido, &pm.datos); err != nil {
		return err
	}

	if pm.datos == nil {
		pm.datos = make(map[string]interface{})
	}

	return nil
}

func main() {
	archivoPrivado := filepath.Join(".afarias19", "datos.json")
	pm, err := NuevoPersistenceManager(archivoPrivado)
	if err != nil {
		log.Fatal("No se pudo iniciar el gestor de persistencia: ", err)
	}

	pm.Guardar("nombre", "Afarias19")
	pm.Guardar("version", 1)
	pm.Guardar("activo", true)

	if valor, existe := pm.Obtener("nombre"); existe {
		fmt.Printf("Nombre guardado: %v\n", valor)
	}

	if pm.Existe("activo") {
		fmt.Println("La clave 'activo' existe en el almacenamiento")
	}

	if err := pm.Eliminar("version"); err != nil {
		log.Fatal("No se pudo eliminar la clave: ", err)
	}

	fmt.Println("Claves actuales:")
	for _, clave := range pm.Claves() {
		fmt.Println(" -", clave)
	}

	if err := pm.Guardar("estado", "privado_local"); err != nil {
		log.Fatal("No se pudo guardar el estado final: ", err)
	}

	fmt.Printf("Archivo privado: %s\n", archivoPrivado)
	fmt.Println("Proyecto listo para usar de forma local y privada.")
}
