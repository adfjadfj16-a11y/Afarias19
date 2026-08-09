# Pilar 4 — Integridad Verificable (Auto-test al arranque)

## Problema

Cuando el binario falla tras la instalación, el usuario recibe logs crípticos o simplemente silencio. Diagnosticar si el problema es un asset faltante, una configuración incorrecta o un entorno incompatible requiere experiencia técnica avanzada.

## Solución

Implementar un **modo de auto-verificación** (`--self-check`) que el binario puede ejecutar (y que el daemon ejecuta automáticamente al arrancar) para detectar y reportar proactivamente cualquier problema de configuración o integridad antes de iniciar la indexación.

---

## Uso

```bash
# Verificación manual explícita
codebase-memory-mcp --self-check

# El daemon ejecuta self-check automáticamente al arrancar
# (puede deshabilitarse con --skip-self-check para entornos controlados)
codebase-memory-mcp daemon
```

---

## Verificaciones implementadas

### 1. Assets requeridos

Verifica que todos los archivos de assets (UI, modelos, etc.) estén presentes y no corruptos.

```
[PASS] UI assets directory: /usr/local/share/codebase-memory-mcp/ui
[PASS] Asset: index.html (sha256: abc123...)
[PASS] Asset: main.js (sha256: def456...)
[FAIL] Asset: worker.wasm — NOT FOUND
```

### 2. Integridad de checksums

Compara los checksums de los assets contra el manifiesto embebido en el binario (generado en tiempo de compilación).

```go
// assets_manifest.go (generado por build script)
var AssetsManifest = map[string]string{
    "ui/index.html": "sha256:abc123...",
    "ui/main.js":    "sha256:def456...",
    "ui/worker.wasm": "sha256:789ghi...",
}
```

### 3. Configuración del entorno

```
[PASS] Config file: ~/.config/codebase-memory-mcp/config.toml
[PASS] Socket directory writable: /run/user/1000/
[PASS] Port available: 9090
[WARN] max_memory_budget_mb not set — using unlimited (recommended: set a limit)
[PASS] Go runtime version: compatible
```

### 4. Permisos y accesibilidad

```
[PASS] Binary is executable
[PASS] Binary signature: valid (EV certificate / Developer ID)
[FAIL] Config directory not writable: /etc/codebase-memory-mcp/
       → Fix: chown $USER /etc/codebase-memory-mcp/ or use --config ~/.config/...
```

---

## Formato de salida

### Salida legible (por defecto)

```
codebase-memory-mcp self-check v1.x.x
══════════════════════════════════════
Checking assets...          [4/4 PASS]
Checking checksums...       [4/4 PASS]
Checking configuration...   [3/4 PASS] 1 warning
Checking permissions...     [2/3 PASS] 1 failure

RESULT: NOT READY — 1 failure detected

Failures:
  ✗ Config directory not writable: /etc/codebase-memory-mcp/
    Fix: sudo chown $USER /etc/codebase-memory-mcp/
         or start with --config ~/.config/codebase-memory-mcp/config.toml

Warnings:
  ⚠ max_memory_budget_mb not set
    Recommended: add max_memory_budget_mb = 512 to your config

Run with --self-check --json for machine-readable output.
```

### Salida JSON (`--self-check --json`)

```json
{
  "version": "1.x.x",
  "result": "not_ready",
  "checks": [
    { "category": "assets", "passed": 4, "failed": 0, "warnings": 0 },
    { "category": "checksums", "passed": 4, "failed": 0, "warnings": 0 },
    { "category": "configuration", "passed": 3, "failed": 0, "warnings": 1,
      "warnings_detail": ["max_memory_budget_mb not set"] },
    { "category": "permissions", "passed": 2, "failed": 1, "warnings": 0,
      "failures_detail": [
        { "message": "Config directory not writable: /etc/codebase-memory-mcp/",
          "fix": "sudo chown $USER /etc/codebase-memory-mcp/" }
      ]
    }
  ]
}
```

---

## Integración con los tests existentes (`tests/`)

Los contratos de prueba existentes (`test_release_archive_extractor_contract.sh`, `test_smoke_fixture_contract.sh`, etc.) sirven como base para las verificaciones de arranque:

- Las mismas aserciones que los tests validan en CI, el binario las ejecuta en el host del usuario
- **Venue-parity garantizada**: si pasa en CI, debe pasar en el entorno del usuario; si falla, `--self-check` lo detecta

```bash
# tests/test_self_check_contract.sh
# Verifica que --self-check detecta correctamente:
# - asset faltante → exit code 1, mensaje claro
# - checksum incorrecto → exit code 1, archivo identificado
# - configuración válida → exit code 0
# - entorno limpio → exit code 0, "READY"
```

---

## Códigos de salida

| Código | Significado |
|--------|-------------|
| `0` | Todo correcto — listo para arrancar |
| `1` | Uno o más fallos — no arranca |
| `2` | Solo advertencias — arranca con limitaciones |

---

## Comportamiento del daemon al arrancar

```
$ codebase-memory-mcp daemon
Running self-check... [PASS] All checks passed.
Starting daemon on /run/user/1000/cbmcp.sock
Listening for agent connections...
```

Si hay fallos:
```
$ codebase-memory-mcp daemon
Running self-check... [FAIL]
  ✗ Asset worker.wasm not found
Daemon startup aborted. Run --self-check for details.
```
