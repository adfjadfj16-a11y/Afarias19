// Package main es el punto de entrada del programa.
// Aquí definimos todo lo relacionado con guardar y cargar datos en un archivo.
package main

import (
	"bytes"         // nos permite trabajar con buffers de bytes en memoria
	"compress/gzip" // nos permite comprimir y descomprimir datos en formato gzip
	"encoding/json" // nos permite convertir datos Go a formato JSON y viceversa
	"errors"        // nos permite crear mensajes de error personalizados
	"fmt"           // nos permite imprimir mensajes con formato en pantalla
	"io"            // nos permite leer y escribir flujos de datos
	"log"           // nos permite registrar errores graves y detener el programa de forma segura
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

// Existe verifica si una clave está guardada, sin necesidad de obtener su valor.
// Útil cuando solo quieres saber si algo existe antes de buscarlo.
func (pm *PersistenceManager) Existe(clave string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	_, existe := pm.datos[clave]
	return existe
}

// LimpiarTodo borra todos los datos guardados, tanto en memoria como en el archivo.
// Úsalo con cuidado: no se puede deshacer.
func (pm *PersistenceManager) LimpiarTodo() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Vaciamos el mapa creando uno nuevo vacío
	pm.datos = make(map[string]interface{})

	// Guardamos el estado vacío en el archivo
	return pm.persistir()
}

// persistir escribe el contenido del mapa en el archivo como JSON comprimido con gzip.
// Usa escritura atómica: primero escribe en un archivo temporal y luego lo renombra.
// La compresión gzip reduce el tamaño del archivo considerablemente, especialmente con
// muchas entradas o valores de texto largos.
func (pm *PersistenceManager) persistir() error {
	// Convertimos el mapa a JSON compacto (sin sangría: el archivo ya va comprimido)
	contenidoJSON, err := json.Marshal(pm.datos)
	if err != nil {
		return err
	}

	// Comprimimos el JSON usando gzip en memoria
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := gz.Write(contenidoJSON); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	// Escribimos primero en un archivo temporal (seguridad ante cortes de luz o cierres bruscos)
	rutaTemporal := pm.rutaArchivo + ".tmp"
	if err := os.WriteFile(rutaTemporal, buf.Bytes(), 0644); err != nil {
		return err
	}

	// Renombramos el temporal al archivo definitivo (operación atómica en la mayoría de sistemas)
	return os.Rename(rutaTemporal, pm.rutaArchivo)
}

// cargar lee el archivo y carga los datos en memoria.
// Detecta automáticamente si el archivo está comprimido con gzip o es JSON plano,
// para mantener compatibilidad con archivos creados antes de añadir compresión.
func (pm *PersistenceManager) cargar() error {
	// Leemos todo el contenido del archivo
	contenido, err := os.ReadFile(pm.rutaArchivo)
	if err != nil {
		return err // puede ser os.ErrNotExist si el archivo no existe
	}

	// Intentamos descomprimir con gzip.
	// Solo si el encabezado no es gzip (ErrHeader) caemos al JSON plano,
	// para mantener compatibilidad con archivos anteriores a la compresión.
	// Cualquier otro error se devuelve directamente.
	gr, err := gzip.NewReader(bytes.NewReader(contenido))
	if err != nil && !errors.Is(err, gzip.ErrHeader) {
		return err
	}
	if err == nil {
		defer gr.Close()
		descomprimido, err := io.ReadAll(gr)
		if err != nil {
			return err
		}
		contenido = descomprimido
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
		// log.Fatal es más seguro que panic: muestra el error y cierra el programa limpiamente
		log.Fatal("No se pudo iniciar el gestor de persistencia: ", err)
	}

	// Guardamos algunos valores de ejemplo
	_ = pm.Guardar("nombre", "Afarias19")
	_ = pm.Guardar("version", 1)
	_ = pm.Guardar("activo", true)

	// Recuperamos y mostramos un valor
	if valor, existe := pm.Obtener("nombre"); existe {
		fmt.Printf("Nombre guardado: %v\n", valor)
	}

	// Verificamos si una clave existe sin necesidad de obtener su valor
	if pm.Existe("activo") {
		fmt.Println("La clave 'activo' existe en el almacenamiento")
	}

	// Eliminamos una clave
	_ = pm.Eliminar("version")

	// Mostramos todas las claves que quedan
	fmt.Println("Claves actuales:")
	for _, clave := range pm.Claves() {
		fmt.Println(" -", clave)
	}
}
