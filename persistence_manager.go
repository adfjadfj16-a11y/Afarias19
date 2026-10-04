// Package main es el punto de entrada del programa.
// Aquí definimos todo lo relacionado con guardar y cargar datos en un archivo.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// PersistenceManager guarda datos en un archivo JSON local y privado.
type PersistenceManager struct {
	rutaArchivo string
	datos       map[string]interface{}
	mu          sync.RWMutex
}

// NuevoPersistenceManager crea un gestor de almacenamiento local privado.
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

type AppState struct {
	Nombre string   `json:"nombre"`
	Email  string   `json:"email"`
	Ciudad string   `json:"ciudad"`
	Notas  []string `json:"notas"`
}

type App struct {
	pm    *PersistenceManager
	state AppState
}

func NewApp(rutaArchivo string) (*App, error) {
	pm, err := NuevoPersistenceManager(rutaArchivo)
	if err != nil {
		return nil, err
	}

	app := &App{pm: pm}
	if err := app.cargarEstado(); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *App) cargarEstado() error {
	valor, existe := a.pm.Obtener("app_state")
	if !existe {
		a.state = AppState{Notas: []string{}}
		return nil
	}

	bytes, err := json.Marshal(valor)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &a.state); err != nil {
		return err
	}
	if a.state.Notas == nil {
		a.state.Notas = []string{}
	}
	return nil
}

func (a *App) guardarEstado() error {
	return a.pm.Guardar("app_state", a.state)
}

func (a *App) mostrarMenu() {
	fmt.Println("\n=== Afarias19 App Privada ===")
	fmt.Println("1. Definir nombre")
	fmt.Println("2. Definir email")
	fmt.Println("3. Definir ciudad")
	fmt.Println("4. Añadir nota")
	fmt.Println("5. Ver notas")
	fmt.Println("6. Ver perfil")
	fmt.Println("7. Limpiar notas")
	fmt.Println("8. Reiniciar todo")
	fmt.Println("0. Salir")
	fmt.Print("Selecciona una opción: ")
}

func (a *App) leerTexto(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	texto, _ := reader.ReadString('\n')
	return strings.TrimSpace(texto)
}

func (a *App) ejecutar() error {
	scanner := bufio.NewReader(os.Stdin)
	for {
		a.mostrarMenu()
		entrada, err := scanner.ReadString('\n')
		if err != nil {
			if errors.Is(err, os.ErrClosed) {
				break
			}
			return err
		}

		opcion := strings.TrimSpace(entrada)
		switch opcion {
		case "1":
			a.state.Nombre = a.leerTexto("Nombre: ")
			if err := a.guardarEstado(); err != nil {
				fmt.Println("Error guardando nombre:", err)
			}
		case "2":
			a.state.Email = a.leerTexto("Email: ")
			if err := a.guardarEstado(); err != nil {
				fmt.Println("Error guardando email:", err)
			}
		case "3":
			a.state.Ciudad = a.leerTexto("Ciudad: ")
			if err := a.guardarEstado(); err != nil {
				fmt.Println("Error guardando ciudad:", err)
			}
		case "4":
			nota := a.leerTexto("Escribe la nota: ")
			if nota == "" {
				fmt.Println("La nota no puede estar vacía.")
				continue
			}
			a.state.Notas = append(a.state.Notas, nota)
			if err := a.guardarEstado(); err != nil {
				fmt.Println("Error guardando nota:", err)
			}
			fmt.Println("Nota guardada.")
		case "5":
			if len(a.state.Notas) == 0 {
				fmt.Println("No hay notas guardadas.")
				continue
			}
			fmt.Println("\nNotas:")
			for i, nota := range a.state.Notas {
				fmt.Printf("%d. %s\n", i+1, nota)
			}
		case "6":
			fmt.Println("\nPerfil actual:")
			fmt.Printf("- Nombre: %s\n", a.state.Nombre)
			fmt.Printf("- Email: %s\n", a.state.Email)
			fmt.Printf("- Ciudad: %s\n", a.state.Ciudad)
			fmt.Printf("- Notas: %d\n", len(a.state.Notas))
		case "7":
			a.state.Notas = []string{}
			if err := a.guardarEstado(); err != nil {
				fmt.Println("Error al limpiar notas:", err)
			} else {
				fmt.Println("Notas limpiadas.")
			}
		case "8":
			a.state = AppState{Notas: []string{}}
			if err := a.guardarEstado(); err != nil {
				fmt.Println("Error al reiniciar:", err)
			} else {
				fmt.Println("Se reinició el contenido local.")
			}
		case "0", "salir", "exit":
			fmt.Println("Gracias por usar Afarias19 App Privada.")
			return nil
		default:
			fmt.Println("Opción no válida. Intenta de nuevo.")
		}
	}
	return nil
}

func main() {
	archivoPrivado := filepath.Join(".afarias19", "datos.json")
	app, err := NewApp(archivoPrivado)
	if err != nil {
		log.Fatal("No se pudo iniciar la app privada: ", err)
	}

	fmt.Println("Archivo privado activo:", archivoPrivado)
	if err := app.ejecutar(); err != nil {
		log.Fatal("Error en la app: ", err)
	}
}
