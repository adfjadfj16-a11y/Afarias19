# Pilar 3 — Observabilidad y Presupuesto de Memoria

## Problema

Sin límites explícitos de memoria, el daemon puede consumir RAM de forma irrestricta y causar caídas del sistema host. En entornos empresariales con múltiples agentes y codebases grandes, esto es inaceptable.

## Solución

Implementar un sistema de **presupuesto de memoria** configurable con monitoreo en tiempo real y métricas observables.

---

## Flag `--max-memory-budget`

### Uso

```bash
# Limitar el daemon a 512 MB de RAM
codebase-memory-mcp daemon --max-memory-budget 512

# En el archivo de configuración (config.toml / config.json)
[daemon]
max_memory_budget_mb = 512
```

### Comportamiento cuando se supera el límite

1. **Advertencia** (al 80% del límite): emite log estructurado de advertencia
2. **Pausa de indexación** (al 95% del límite): detiene nuevas indexaciones, sirve solo consultas del índice existente
3. **Rechazo de nuevos registros** (al 100%): nuevos agentes reciben error `MEMORY_BUDGET_EXCEEDED` hasta que baje la presión

```json
// Log estructurado al superar el 80%
{
  "level": "warn",
  "event": "memory_budget_pressure",
  "used_mb": 412,
  "budget_mb": 512,
  "utilization_pct": 80.5,
  "action": "indexing_throttled",
  "timestamp": "2026-08-09T17:00:00Z"
}
```

---

## Monitoreo en tiempo real

El daemon expone métricas básicas mediante dos mecanismos:

### 1. Log periódico (cada 60s por defecto)

```json
{
  "level": "info",
  "event": "daemon_metrics",
  "memory_used_mb": 128,
  "memory_budget_mb": 512,
  "agents_connected": 3,
  "indexes_loaded": 2,
  "index_total_size_mb": 95,
  "uptime_seconds": 3600
}
```

### 2. Endpoint HTTP de health-check (`/health`)

```bash
# Habilitado con --metrics-port (opcional, deshabilitado por defecto)
codebase-memory-mcp daemon --metrics-port 9090
```

```
GET http://localhost:9090/health
```

```json
{
  "status": "ok",
  "memory": {
    "used_mb": 128,
    "budget_mb": 512,
    "utilization_pct": 25.0,
    "pressure": "normal"
  },
  "agents": {
    "connected": 3
  },
  "indexes": {
    "loaded": 2,
    "total_size_mb": 95
  },
  "uptime_seconds": 3600
}
```

Estados posibles de `status`: `"ok"` | `"degraded"` | `"overloaded"`
Estados posibles de `pressure`: `"normal"` | `"elevated"` | `"critical"`

---

## Implementación interna

```go
type MemoryBudget struct {
    LimitMB    int64
    mu         sync.Mutex
    currentMB  int64
}

func (b *MemoryBudget) Check() BudgetStatus {
    b.mu.Lock()
    defer b.mu.Unlock()
    pct := float64(b.currentMB) / float64(b.LimitMB) * 100
    switch {
    case pct >= 100:
        return BudgetExceeded
    case pct >= 95:
        return BudgetCritical
    case pct >= 80:
        return BudgetElevated
    default:
        return BudgetNormal
    }
}
```

---

## Integración con sistemas de monitoreo externos

El endpoint `/health` es compatible con:

- **Kubernetes**: liveness/readiness probes
- **Docker**: `HEALTHCHECK` instruction
- **Prometheus**: con un exporter simple sobre las métricas JSON
- **Uptime Kuma / Grafana**: monitoreo HTTP genérico

```dockerfile
# Ejemplo Docker
HEALTHCHECK --interval=30s --timeout=5s \
  CMD curl -f http://localhost:9090/health || exit 1
```

---

## Configuración completa de referencia

```toml
[daemon]
max_memory_budget_mb = 512    # 0 = sin límite
metrics_port = 9090           # 0 = deshabilitado
metrics_interval_seconds = 60
memory_warning_pct = 80
memory_pause_pct = 95
index_ttl_seconds = 300       # TTL del índice tras último agente desconectado
```
