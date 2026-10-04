// Package main es el punto de entrada del programa.
// Aquí definimos todo lo relacionado con guardar y cargar datos en un archivo.
package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const defaultPassword = "afarias19"

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

type ConfigPrivada struct {
	PasswordHash string    `json:"password_hash"`
	PasswordSalt string    `json:"password_salt"`
	Datos        AppState  `json:"datos"`
}

type App struct {
	pm      *PersistenceManager
	config  ConfigPrivada
	storage map[string]interface{}
}

func randomSalt() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func hashPassword(password, salt string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

func NewApp(rutaArchivo string) (*App, error) {
	pm, err := NuevoPersistenceManager(rutaArchivo)
	if err != nil {
		return nil, err
	}

	app := &App{pm: pm, storage: map[string]interface{}{}}
	if err := app.cargarConfig(); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *App) cargarConfig() error {
	if valor, existe := a.pm.Obtener("private_app"); existe {
		bytes, err := json.Marshal(valor)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(bytes, &a.config); err != nil {
			return err
		}
	} else {
		a.config = ConfigPrivada{
			PasswordSalt: randomSalt(),
			Datos:        AppState{Notas: []string{}},
		}
		a.config.PasswordHash = hashPassword(defaultPassword, a.config.PasswordSalt)
	}

	if a.config.Datos.Notas == nil {
		a.config.Datos.Notas = []string{}
	}
	return nil
}

func (a *App) guardarConfig() error {
	return a.pm.Guardar("private_app", a.config)
}

func (a *App) validarPassword(password string) bool {
	if a.config.PasswordHash == "" {
		return password == defaultPassword
	}
	return hashPassword(password, a.config.PasswordSalt) == a.config.PasswordHash
}

func (a *App) setPassword(newPassword string) error {
	if strings.TrimSpace(newPassword) == "" {
		return errors.New("la contraseña no puede estar vacía")
	}
	a.config.PasswordSalt = randomSalt()
	a.config.PasswordHash = hashPassword(newPassword, a.config.PasswordSalt)
	return a.guardarConfig()
}

func (a *App) mostrarMenu() {
	fmt.Println("\n=== Afarias19 App Privada ===")
	fmt.Println("1. Definir nombre")
	fmt.Println("2. Definir email")
	fmt.Println("3. Definir ciudad")
	fmt.Println("4. Añadir nota")
	fmt.Println("5. Ver notas")
	fmt.Println("6. Ver perfil")
	fmt.Println("7. Cambiar contraseña")
	fmt.Println("8. Exportar datos")
	fmt.Println("9. Importar datos")
	fmt.Println("10. Reiniciar todo")
	fmt.Println("0. Salir")
	fmt.Print("Selecciona una opción: ")
}

func (a *App) leerTexto(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	texto, _ := reader.ReadString('\n')
	return strings.TrimSpace(texto)
}

func (a *App) exportarDatos() error {
	bytes, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return err
	}
	archivo := filepath.Join(".afarias19", "export-afarias19.json")
	return os.WriteFile(archivo, bytes, 0o600)
}

func (a *App) importarDatos(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var nueva ConfigPrivada
	if err := json.Unmarshal(bytes, &nueva); err != nil {
		return err
	}
	if nueva.Datos.Notas == nil {
		nueva.Datos.Notas = []string{}
	}
	a.config = nueva
	return a.guardarConfig()
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
			a.config.Datos.Nombre = a.leerTexto("Nombre: ")
			if err := a.guardarConfig(); err != nil {
				fmt.Println("Error guardando nombre:", err)
			}
		case "2":
			a.config.Datos.Email = a.leerTexto("Email: ")
			if err := a.guardarConfig(); err != nil {
				fmt.Println("Error guardando email:", err)
			}
		case "3":
			a.config.Datos.Ciudad = a.leerTexto("Ciudad: ")
			if err := a.guardarConfig(); err != nil {
				fmt.Println("Error guardando ciudad:", err)
			}
		case "4":
			nota := a.leerTexto("Escribe la nota: ")
			if nota == "" {
				fmt.Println("La nota no puede estar vacía.")
				continue
			}
			a.config.Datos.Notas = append(a.config.Datos.Notas, nota)
			if err := a.guardarConfig(); err != nil {
				fmt.Println("Error guardando nota:", err)
			}
			fmt.Println("Nota guardada.")
		case "5":
			if len(a.config.Datos.Notas) == 0 {
				fmt.Println("No hay notas guardadas.")
				continue
			}
			fmt.Println("\nNotas:")
			for i, nota := range a.config.Datos.Notas {
				fmt.Printf("%d. %s\n", i+1, nota)
			}
		case "6":
			fmt.Println("\nPerfil actual:")
			fmt.Printf("- Nombre: %s\n", a.config.Datos.Nombre)
			fmt.Printf("- Email: %s\n", a.config.Datos.Email)
			fmt.Printf("- Ciudad: %s\n", a.config.Datos.Ciudad)
			fmt.Printf("- Notas: %d\n", len(a.config.Datos.Notas))
		case "7":
			password := a.leerTexto("Nueva contraseña: ")
			if err := a.setPassword(password); err != nil {
				fmt.Println("Error cambiando contraseña:", err)
			} else {
				fmt.Println("Contraseña actualizada.")
			}
		case "8":
			if err := a.exportarDatos(); err != nil {
				fmt.Println("Error exportando datos:", err)
			} else {
				fmt.Println("Exportación creada en .afarias19/export-afarias19.json")
			}
		case "9":
			path := a.leerTexto("Ruta del archivo JSON a importar: ")
			if path == "" {
				fmt.Println("Debes indicar una ruta válida.")
				continue
			}
			if err := a.importarDatos(path); err != nil {
				fmt.Println("Error importando datos:", err)
			} else {
				fmt.Println("Datos importados correctamente.")
			}
		case "10":
			a.config = ConfigPrivada{
				PasswordSalt: randomSalt(),
				PasswordHash: hashPassword(defaultPassword, randomSalt()),
				Datos:        AppState{Notas: []string{}},
			}
			a.config.PasswordHash = hashPassword(defaultPassword, a.config.PasswordSalt)
			if err := a.guardarConfig(); err != nil {
				fmt.Println("Error reiniciando datos:", err)
			} else {
				fmt.Println("Se reinició el contenido local y la contraseña vuelve a ser la predeterminada.")
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

func renderLoginPage(message string) string {
	return `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Afarias19 - Acceso privado</title>
  <style>
    :root {
      --bg: #09111d;
      --panel: rgba(17, 25, 40, 0.85);
      --panel-strong: rgba(12,18,30,0.98);
      --accent: #49d4ff;
      --accent-2: #7c5cff;
      --text: #edf6ff;
      --muted: #9bb3c9;
      --success: #4ade80;
      --warning: #fbbf24;
      --danger: #f87171;
      --shadow: rgba(73, 212, 255, 0.35);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      background: radial-gradient(circle at top left, rgba(124,92,255,0.25), transparent 30%),
                  radial-gradient(circle at bottom right, rgba(73,212,255,0.18), transparent 30%),
                  var(--bg);
      color: var(--text);
      font-family: Inter, Segoe UI, sans-serif;
    }
    .card {
      width: min(420px, 92vw);
      background: var(--panel);
      border: 1px solid rgba(255,255,255,0.08);
      border-radius: 24px;
      box-shadow: 0 0 35px var(--shadow);
      padding: 30px 26px;
      backdrop-filter: blur(16px);
    }
    h1 { margin: 0 0 10px; font-size: 28px; }
    .subtitle { color: var(--muted); margin-bottom: 22px; }
    form { display: grid; gap: 14px; }
    label { display: grid; gap: 8px; font-weight: 600; }
    input {
      border-radius: 12px; border: 1px solid rgba(255,255,255,0.12);
      background: rgba(255,255,255,0.04);
      color: var(--text);
      padding: 12px 14px;
      font-size: 16px;
    }
    button {
      margin-top: 12px; border: none; border-radius: 12px; padding: 12px 16px;
      font-weight: 700; cursor: pointer;
      background: linear-gradient(135deg, var(--accent), var(--accent-2));
      color: #06111d; box-shadow: 0 12px 20px rgba(73,212,255,0.18);
    }
    .message {
      background: rgba(248, 113, 113, 0.12); color: #ffd9d9; border: 1px solid rgba(248,113,113,0.3);
      border-radius: 10px; padding: 10px 12px; font-size: 14px;
    }
    .footer { margin-top: 14px; color: var(--muted); text-align: center; font-size: 12px; }
  </style>
</head>
<body>
  <div class="card">
    <h1>Afarias19</h1>
    <div class="subtitle">Acceso privado local</div>
    {{if .Message}}<div class="message">{{.Message}}</div>{{end}}
    <form method="POST" action="/login">
      <label>
        Contraseña
        <input type="password" name="password" placeholder="Introduce tu contraseña" required>
      </label>
      <button type="submit">Entrar</button>
    </form>
    <div class="footer">Tu información se guarda localmente en este equipo.</div>
  </div>
</body>
</html>`
}

func renderDashboardPage(data ConfigPrivada) string {
	return `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Afarias19 Dashboard</title>
  <style>
    :root {
      --bg: #07111d;
      --panel: rgba(14,22,35,0.9);
      --panel-soft: rgba(20,29,42,0.9);
      --border: rgba(255,255,255,0.08);
      --text: #edf6ff;
      --muted: #9bb3c9;
      --accent: #49d4ff;
      --accent-2: #8b5cf6;
      --success: #34d399;
      --warning: #fbbf24;
      --danger: #f87171;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0; min-height: 100vh; background: radial-gradient(circle at top, rgba(139,92,246,0.16), transparent 30%), #07111d; color: var(--text);
      font-family: Inter, Segoe UI, sans-serif; padding: 30px;
    }
    .wrap { max-width: 1100px; margin: 0 auto; }
    .top {
      display: flex; justify-content: space-between; align-items: center; gap: 16px; margin-bottom: 22px;
    }
    .brand { font-size: 32px; font-weight: 800; }
    .actions { display: flex; gap: 10px; flex-wrap: wrap; }
    .btn {
      border: 1px solid var(--border); background: rgba(255,255,255,0.04); color: var(--text); padding: 10px 14px; border-radius: 12px; cursor: pointer; font-weight: 700;
    }
    .btn.primary { background: linear-gradient(135deg, var(--accent), var(--accent-2)); color: #06111d; border: none; }
    .btn.danger { background: rgba(248,113,113,0.12); color: #ffd9d9; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 20px; }
    .card {
      background: var(--panel); border: 1px solid var(--border); border-radius: 20px; padding: 20px; box-shadow: 0 12px 26px rgba(0,0,0,0.18);
    }
    .label { color: var(--muted); font-size: 12px; letter-spacing: 0.1em; text-transform: uppercase; }
    input, textarea {
      width: 100%; border: 1px solid var(--border); background: rgba(255,255,255,0.04); color: var(--text); border-radius: 12px; padding: 11px 12px; margin-top: 8px; font-size: 15px;
    }
    textarea { min-height: 120px; resize: vertical; }
    form { display: grid; gap: 12px; }
    .note-list { list-style: none; margin: 0; padding: 0; display: grid; gap: 10px; }
    .note-item {
      background: rgba(255,255,255,0.02); border: 1px solid var(--border); border-radius: 12px; padding: 10px 12px; color: var(--text);
    }
    .small { color: var(--muted); font-size: 13px; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="top">
      <div class="brand">Afarias19</div>
      <div class="actions">
        <a href="/export" class="btn primary" download>Exportar</a>
        <form id="importForm" enctype="multipart/form-data" method="POST" action="/import" style="display:inline;">
          <label class="btn" for="fileInput" style="display:inline-block; cursor:pointer;">Importar</label>
          <input id="fileInput" type="file" name="file" accept="application/json" style="display:none;">
        </form>
        <form method="POST" action="/logout" style="display:inline;">
          <button class="btn danger" type="submit">Cerrar sesión</button>
        </form>
      </div>
    </div>

    <div class="grid">
      <div class="card">
        <div class="label">Perfil</div>
        <form method="POST" action="/save">
          <div>
            <label>Nombre</label>
            <input name="nombre" value="{{.Datos.Nombre}}">
          </div>
          <div>
            <label>Email</label>
            <input name="email" value="{{.Datos.Email}}">
          </div>
          <div>
            <label>Ciudad</label>
            <input name="ciudad" value="{{.Datos.Ciudad}}">
          </div>
          <button class="btn primary" type="submit">Guardar perfil</button>
        </form>
      </div>

      <div class="card">
        <div class="label">Notas</div>
        <form method="POST" action="/note">
          <div>
            <label>Nueva nota</label>
            <textarea name="nota" placeholder="Escribe la nota privada..."></textarea>
          </div>
          <button class="btn primary" type="submit">Añadir nota</button>
        </form>
        <div style="margin-top:16px;">
          <div class="small">Notas guardadas: {{len .Datos.Notas}}</div>
          <ul class="note-list">
            {{range .Datos.Notas}}
            <li class="note-item">{{.}}</li>
            {{else}}
            <li class="note-item">No hay notas guardadas.</li>
            {{end}}
          </ul>
        </div>
      </div>

      <div class="card">
        <div class="label">Privacidad</div>
        <form method="POST" action="/password">
          <div>
            <label>Nueva contraseña</label>
            <input type="password" name="password" placeholder="Escribe una contraseña">
          </div>
          <button class="btn primary" type="submit">Actualizar contraseña</button>
        </form>
        <form method="POST" action="/reset" style="margin-top:20px;">
          <button class="btn danger" type="submit">Reiniciar todo</button>
        </form>
      </div>
    </div>
  </div>
<script>
  const fileInput = document.getElementById('fileInput');
  fileInput && fileInput.addEventListener('change', () => {
    if (fileInput.files && fileInput.files[0]) {
      const form = document.getElementById('importForm');
      form && form.submit();
    }
  });
</script>
</body>
</html>`
}

func mustHTML(value string) template.HTML {
	return template.HTML(value)
}

func (a *App) runHTTP(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if !a.isAuthenticated(r) {
			_, _ = io.WriteString(w, renderLoginPage(""))
			return
		}
		_, _ = io.WriteString(w, renderDashboardPage(a.config))
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}
		password := r.FormValue("password")
		if a.validarPassword(password) {
			http.SetCookie(w, &http.Cookie{Name: "afarias19_session", Value: "authed", Path: "/", HttpOnly: true})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		_, _ = io.WriteString(w, renderLoginPage("Contraseña incorrecta."))
	})

	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "afarias19_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !a.isAuthenticated(r) {
			http.Error(w, "No autorizado", http.StatusUnauthorized)
			return
		}
		a.config.Datos.Nombre = r.FormValue("nombre")
		a.config.Datos.Email = r.FormValue("email")
		a.config.Datos.Ciudad = r.FormValue("ciudad")
		if err := a.guardarConfig(); err != nil {
			http.Error(w, "Error guardando perfil", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("/note", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !a.isAuthenticated(r) {
			http.Error(w, "No autorizado", http.StatusUnauthorized)
			return
		}
		nota := strings.TrimSpace(r.FormValue("nota"))
		if nota == "" {
			http.Error(w, "La nota no puede estar vacía", http.StatusBadRequest)
			return
		}
		a.config.Datos.Notas = append(a.config.Datos.Notas, nota)
		if err := a.guardarConfig(); err != nil {
			http.Error(w, "Error guardando nota", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("/password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !a.isAuthenticated(r) {
			http.Error(w, "No autorizado", http.StatusUnauthorized)
			return
		}
		if err := a.setPassword(r.FormValue("password")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !a.isAuthenticated(r) {
			http.Error(w, "No autorizado", http.StatusUnauthorized)
			return
		}
		a.config = ConfigPrivada{
			PasswordSalt: randomSalt(),
			Datos: AppState{Notas: []string{}},
		}
		a.config.PasswordHash = hashPassword(defaultPassword, a.config.PasswordSalt)
		if err := a.guardarConfig(); err != nil {
			http.Error(w, "Error reiniciando datos", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "afarias19_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("/export", func(w http.ResponseWriter, r *http.Request) {
		if !a.isAuthenticated(r) {
			http.Error(w, "No autorizado", http.StatusUnauthorized)
			return
		}
		bytes, err := json.MarshalIndent(a.config, "", "  ")
		if err != nil {
			http.Error(w, "Error exportando", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=afarias19-backup.json")
		_, _ = w.Write(bytes)
	})

	mux.HandleFunc("/import", func(w http.ResponseWriter, r *http.Request) {
		if !a.isAuthenticated(r) {
			http.Error(w, "No autorizado", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Archivo requerido", http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "No se pudo leer el archivo", http.StatusBadRequest)
			return
		}
		var imported ConfigPrivada
		if err := json.Unmarshal(data, &imported); err != nil {
			http.Error(w, "El archivo no es un JSON válido", http.StatusBadRequest)
			return
		}
		if imported.Datos.Notas == nil {
			imported.Datos.Notas = []string{}
		}
		a.config = imported
		if err := a.guardarConfig(); err != nil {
			http.Error(w, "No se pudo guardar la importación", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	fmt.Printf("Servidor privado listo en http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func (a *App) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("afarias19_session")
	if err != nil {
		return false
	}
	return cookie.Value == "authed"
}

func main() {
	archivoPrivado := filepath.Join(".afarias19", "datos.json")
	app, err := NewApp(archivoPrivado)
	if err != nil {
		log.Fatal("No se pudo iniciar la app privada: ", err)
	}

	cliMode := flag.Bool("cli", false, "Usa el modo texto clásico en lugar del navegador local")
	addr := flag.String("addr", ":8080", "Dirección del servidor web")
	flag.Parse()

	if *cliMode {
		fmt.Println("Archivo privado activo:", archivoPrivado)
		if err := app.ejecutar(); err != nil {
			log.Fatal("Error en la app: ", err)
		}
		return
	}

	if err := app.runHTTP(*addr); err != nil {
		log.Fatal("Error iniciando servidor web: ", err)
	}
}
