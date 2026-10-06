# Contribución a Afarias19

¡Gracias por tu interés en contribuir a Afarias19! Este documento describe el proceso de contribución y cómo configurar tu entorno de desarrollo.

## 📋 Requisitos Previos

- **Go 1.24+** — Lenguaje principal para binarios y CLI
- **Python 3.10+** — Bot de trading y tests
- **Node.js 18+** — Herramientas JavaScript (opcional)
- **Git** — Control de versiones
- **GitHub CLI** (opcional pero recomendado) — `gh` para interactuar con GitHub

Verificar versiones:

```bash
go version        # go version go1.24 ...
python3 --version # Python 3.10.0 ...
node --version    # v18.x.x ...
```

## 🔧 Setup Local

### 1. Fork y Clone

```bash
# Fork en GitHub (botón "Fork")
git clone https://github.com/TU_USERNAME/Afarias19.git
cd Afarias19
git remote add upstream https://github.com/adfjadfj16-a11y/Afarias19.git
```

### 2. Instalar Dependencias

**Go:**
```bash
go mod download
go mod tidy
```

**Python:**
```bash
pip install -r requirements.txt  # Si existe
pip install pytest pytest-cov black flake8
```

**JavaScript:**
```bash
npm install  # Si existe package.json
npm install -D eslint eslint-config-standard
```

### 3. Verificar Setup

```bash
# Test Go
go test -v ./...

# Test Python
python3 -m pytest tests/ -v

# Build self-check
go build -o ./afarias19-test ./cmd/self-check
./afarias19-test --self-check
```

## 📝 Flujo de Trabajo

### 1. Crear Branch para tu Feature

```bash
# Sincronizar con upstream
git fetch upstream
git checkout -b feature/tu-feature upstream/main

# O para fixes
git checkout -b fix/descripcion-bug upstream/main
```

### 2. Hacer Cambios

Mantén los cambios **focalizados** y **atómicos** (un cambio lógico = un commit).

**Convenciones de commit:**

```
feat: Agregar verificación de assets en self-check
fix: Corregir race condition en persistence_manager
docs: Actualizar documentación de Pilar 3
test: Agregar tests para checkConfig()
refactor: Simplificar lógica de indexación
chore: Actualizar dependencias de Go
```

### 3. Lint & Format

```bash
# Go
go fmt ./...
golangci-lint run ./...

# Python
black *.py tests/
flake8 *.py tests/

# JavaScript
npx eslint *.js
```

### 4. Tests

```bash
# Go (con coverage)
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Python
pytest tests/ -v --cov=.

# Benchmark (si aplica)
go test -bench=. ./cmd/self-check
```

### 5. Push & Pull Request

```bash
git push origin feature/tu-feature
```

En GitHub, abre un Pull Request:

- **Título:** Descriptivo y conciso (ej: "Implement Pilar 3 memory budget")
- **Descripción:** 
  - Qué problema resuelve
  - Cómo lo resuelve
  - Tests incluidos
  - Cambios breaking (si aplica)

**Template:**

```markdown
## Descripción

Breve resumen de qué se implementa.

## Tipo de cambio

- [ ] Bug fix
- [ ] Nuevo feature
- [ ] Breaking change
- [ ] Documentación

## Cambios

- Cambio 1
- Cambio 2

## Tests

- [ ] Tests unitarios agregados
- [ ] Tests de integración pasados
- [ ] Cobertura ≥ 80%

## Checklist

- [ ] Lint pasó (`go fmt`, `black`)
- [ ] Tests locales pasaron
- [ ] Documentación actualizada
- [ ] Sin secrets o datos sensibles
```

## 🏛️ Pilares & Cómo Contribuir

### Pilar 1 — Code Signing

Archivos:
- `.github/workflows/release.yml`

Si quieres contribuir:
- Implementar DigiCert KeyLocker (Windows)
- Implementar Apple Notary (macOS)
- Implementar GPG signing (Linux)
- Agregar tests para validar firmas

### Pilar 2 — Daemon Multitenant

Archivos:
- `cmd/daemon/` (a crear)
- `pkg/ipc/` (a crear)

Si quieres contribuir:
- Implementar servidor IPC (Unix socket / Named Pipe)
- JSON-RPC 2.0 protocol
- Agent registry con sincronización
- Tests de concurrencia (N agentes simultáneos)

### Pilar 3 — Presupuesto de Memoria

Archivos:
- `pkg/memory/` (a crear)
- `pkg/metrics/` (a crear)

Si quieres contribuir:
- Presupuesto configurable
- Monitoreo en tiempo real (go-runtime/memory stats)
- Endpoint `/health` HTTP
- Logs estructurados JSON

### Pilar 4 — Auto-test al Arranque

Archivos:
- `cmd/self-check/main.go` ✅ Implementado
- `cmd/self-check/main_test.go` ✅ Implementado

Si quieres contribuir:
- Agregar verificaciones adicionales (checksums SHA256)
- Validar firmas de código
- Tests más robustos
- Integración con CI/CD

---

## 🧪 Testing

### Escritura de Tests

**Convención Go:**

```go
// archivo.go
func Foo(input string) string {
    return "resultado"
}

// archivo_test.go
func TestFoo(t *testing.T) {
    result := Foo("entrada")
    if result != "esperado" {
        t.Errorf("Foo failed: got %s, want esperado", result)
    }
}
```

**Convención Python:**

```python
# module.py
def foo(input_str):
    return "resultado"

# test_module.py
import pytest
from module import foo

def test_foo():
    assert foo("entrada") == "esperado"

def test_foo_error():
    with pytest.raises(ValueError):
        foo(None)
```

### Cobertura Mínima

- Go: ≥ 80%
- Python: ≥ 75%
- Archivos críticos: ≥ 90%

Verificar cobertura:

```bash
go tool cover -html=coverage.out
pytest --cov=. --cov-report=html
```

## 📚 Documentación

Documenta cambios en:

1. **README.md** — Cambios user-facing
2. **docs/pillar-X.md** — Cambios arquitectónicos
3. **Inline comments** — Lógica compleja
4. **CHANGELOG.md** (si existe) — Release notes

**Formato Markdown:**

```markdown
# Título H1

## Sección H2

- Punto 1
- Punto 2

### Subsección H3

```code block```

> Nota o cita importante
```

## 🔐 Seguridad

### No commits de secrets

**NUNCA** commitear:
- API keys o tokens
- Contraseñas
- Certificados privados
- Datos de prueba sensibles

**Usar en su lugar:**
- GitHub Secrets (para CI/CD)
- `.env` local (con `.gitignore`)
- Documentación de cómo configurar

### Escanear secrets antes de push

```bash
# Opción 1: Usar git hooks
# (agregar a .git/hooks/pre-push)

# Opción 2: Herramienta externa
pip install detect-secrets
detect-secrets scan
```

## 🚀 CI/CD Pipeline

Todos los cambios pasan por:

1. **Lint** (golangci-lint, black, ESLint)
2. **Tests** (go test, pytest)
3. **Build** (5 plataformas: linux-amd64/arm64, darwin-amd64/arm64, windows)
4. **Security Scan** (Trivy)
5. **Integration Tests** (self-check validation)

Si alguno falla:
- Revisa los logs en GitHub Actions
- Haz cambios locales
- Commit y push de nuevo

## 📞 Preguntas & Ayuda

- **Issues:** Usa etiquetas: `[pilar-N]`, `[bug]`, `[feature]`, `[question]`
- **Discussions:** Para debates arquitectónicos o diseño
- **Pull Requests:** Para propuestas concretas

---

**¡Gracias por contribuir a Afarias19!** 🙌
