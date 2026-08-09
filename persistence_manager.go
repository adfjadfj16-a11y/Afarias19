// Package main es el punto de entrada del programa.
// Aquí definimos todo lo relacionado con guardar y cargar datos en un archivo.
package main

import (
	"encoding/json" // nos permite convertir datos Go a formato JSON y viceversa
	"errors"        // nos permite crear mensajes de error personalizados
	"os"            // nos permite leer y escribir archivos en el sistema
	"sync"          // nos permite proteger los datos cuando hay acceso simultáneo
)

// PersistenceManager es la estructura principal.
// Se encarga de guardar datos en un archivo JSON y recuperarlos después.
// Piénsalo como una "caja fuerte" que guarda información en el disco.
type PersistenceManager struct {
	// rutaArchivo es la ubicación del archivo donde se guardan los datos
	rutaArchivo string

	// datos es un mapa clave-valor que almacena la información en memoria
	// (como un diccionario: clave → valor)
	datos map[string]interface{}

	// mu es un candado que evita que dos operaciones modifiquen los datos
	// al mismo tiempo y causen errores (concurrencia segura)
	mu sync.RWMutex
}

// NuevoPersistenceManager crea un nuevo gestor de persistencia.
// Recibe la ruta del archivo donde se guardarán los datos.
// Si el archivo ya existe, carga los datos que tenía guardados.
func NuevoPersistenceManager(rutaArchivo string) (*PersistenceManager, error) {
	// Creamos el gestor con la ruta indicada y un mapa vacío
	pm := &PersistenceManager{
		rutaArchivo: rutaArchivo,
		datos:       make(map[string]interface{}),
	}

	// Intentamos cargar los datos desde el archivo (si ya existe)
	if err := pm.cargar(); err != nil && !errors.Is(err, os.ErrNotExist) {
		// Si el error NO es que el archivo no existe, lo devolvemos
		return nil, err
	}

	return pm, nil
}

// Guardar almacena un valor asociado a una clave.
// Es como guardar una anotación en una agenda: clave = tema, valor = contenido.
// Después de guardarlo en memoria, lo escribe también en el archivo.
func (pm *PersistenceManager) Guardar(clave string, valor interface{}) error {
	// Validamos que la clave no esté vacía
	if clave == "" {
		return errors.New("la clave no puede estar vacía")
	}

	// Bloqueamos para escritura (nadie más puede leer o escribir en este momento)
	pm.mu.Lock()
	defer pm.mu.Unlock() // al terminar la función, liberamos el candado automáticamente

	// Guardamos el valor en memoria
	pm.datos[clave] = valor

	// Escribimos todos los datos en el archivo
	return pm.persistir()
}

// Obtener recupera el valor asociado a una clave.
// Devuelve el valor y un indicador (true/false) de si la clave existía.
func (pm *PersistenceManager) Obtener(clave string) (interface{}, bool) {
	// Bloqueamos solo para lectura (varios pueden leer al mismo tiempo)
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	valor, existe := pm.datos[clave]
	return valor, existe
}

// Eliminar borra un valor del almacenamiento usando su clave.
// Si la clave no existe, no hace nada (no es un error).
func (pm *PersistenceManager) Eliminar(clave string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Eliminamos la clave del mapa en memoria
	delete(pm.datos, clave)

	// Actualizamos el archivo con los datos restantes
	return pm.persistir()
}

// Claves devuelve la lista de todas las claves guardadas actualmente.
func (pm *PersistenceManager) Claves() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Creamos una lista del tamaño exacto necesario
	claves := make([]string, 0, len(pm.datos))
	for clave := range pm.datos {
		claves = append(claves, clave)
	}
	return claves
}

// persistir escribe el contenido del mapa en el archivo JSON.
// Este método es interno (letra minúscula), solo lo usa el propio gestor.
func (pm *PersistenceManager) persistir() error {
	// Convertimos el mapa a formato JSON con sangría (más fácil de leer)
	contenido, err := json.MarshalIndent(pm.datos, "", "  ")
	if err != nil {
		return err
	}

	// Escribimos el JSON en el archivo (0644 = el dueño puede leer/escribir, otros solo leer)
	return os.WriteFile(pm.rutaArchivo, contenido, 0644)
}

// cargar lee el archivo JSON y carga los datos en memoria.
// Si el archivo no existe, simplemente no hace nada (empezamos desde cero).
func (pm *PersistenceManager) cargar() error {
	// Leemos todo el contenido del archivo
	contenido, err := os.ReadFile(pm.rutaArchivo)
	if err != nil {
		return err // puede ser os.ErrNotExist si el archivo no existe
	}

	// Convertimos el JSON de vuelta a un mapa Go
	return json.Unmarshal(contenido, &pm.datos)
}

// main es la función principal: se ejecuta cuando arrancamos el programa.
// Sirve como demostración de cómo usar el PersistenceManager.
func main() {
	// Creamos el gestor, usando "datos.json" como archivo de almacenamiento
	pm, err := NuevoPersistenceManager("datos.json")
	if err != nil {
		panic("No se pudo iniciar el gestor de persistencia: " + err.Error())
	}

	// Guardamos algunos valores de ejemplo
	_ = pm.Guardar("nombre", "Afarias19")
	_ = pm.Guardar("version", 1)
	_ = pm.Guardar("activo", true)

	// Recuperamos y mostramos un valor
	if valor, existe := pm.Obtener("nombre"); existe {
		println("Nombre guardado:", valor.(string))
	}

	// Eliminamos una clave
	_ = pm.Eliminar("version")

	// Mostramos todas las claves que quedan
	println("Claves actuales:")
	for _, clave := range pm.Claves() {
		println(" -", clave)
	}
}
