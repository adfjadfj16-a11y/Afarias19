# Pilar 2 — Consolidación del Daemon (Orquestador Multitenant)

## Problema

En el modelo actual, cada agente instancia su propio proceso con su propio índice de memoria. Cuando N agentes trabajan en el mismo codebase, hay N copias del índice en RAM, N procesos de indexación paralelos y duplicidad masiva de recursos.

## Solución

Evolucionar el daemon a un **orquestador central** que gestiona un índice de memoria compartido. Los agentes se conectan al daemon como clientes ligeros via IPC (Unix socket / Named pipe en Windows).

---

## Arquitectura

```
┌─────────────────────────────────────────────────┐
│                  DAEMON CENTRAL                  │
│                                                  │
│  ┌─────────────┐    ┌──────────────────────────┐ │
│  │ Agent Registry│   │   Shared Memory Index    │ │
│  │             │    │                          │ │
│  │ agent-A ────┼───▶│  /path/to/codebase       │ │
│  │ agent-B ────┼───▶│  (single index, shared)  │ │
│  │ agent-C ────┼───▶│                          │ │
│  └─────────────┘    └──────────────────────────┘ │
│                                                  │
│  IPC: Unix socket ($XDG_RUNTIME_DIR/cbmcp.sock)  │
└─────────────────────────────────────────────────┘
         ▲              ▲              ▲
    [Agent A]      [Agent B]      [Agent C]
   (MCP client)  (MCP client)  (MCP client)
```

---

## Protocolo IPC

Comunicación via **JSON-RPC 2.0** sobre Unix socket (Linux/macOS) o Named Pipe (Windows).

### Mensajes del cliente al daemon

```json
// Registro de agente
{ "jsonrpc": "2.0", "method": "agent.register",
  "params": { "agent_id": "uuid-v4", "codebase_path": "/path/to/project" },
  "id": 1 }

// Consulta al índice
{ "jsonrpc": "2.0", "method": "memory.query",
  "params": { "agent_id": "uuid-v4", "query": "...", "limit": 10 },
  "id": 2 }

// Desregistro de agente
{ "jsonrpc": "2.0", "method": "agent.unregister",
  "params": { "agent_id": "uuid-v4" },
  "id": 3 }
```

### Respuestas del daemon

```json
// Registro exitoso
{ "jsonrpc": "2.0", "result": { "status": "registered", "index_ready": true }, "id": 1 }

// Resultado de consulta
{ "jsonrpc": "2.0", "result": { "hits": [...] }, "id": 2 }
```

---

## Registro de sesiones de agente

```go
type AgentRegistry struct {
    mu     sync.RWMutex
    agents map[string]*AgentSession
}

type AgentSession struct {
    ID            string
    CodebasePath  string
    RegisteredAt  time.Time
    LastActiveAt  time.Time
    Conn          net.Conn
}

func (r *AgentRegistry) Register(id, path string, conn net.Conn) error { ... }
func (r *AgentRegistry) Unregister(id string) { ... }
func (r *AgentRegistry) ActiveCount() int { ... }
```

---

## Gestión del índice compartido

- El daemon mantiene **un índice por codebase path** (no uno por agente)
- Al registrarse el primer agente de un codebase, el daemon inicia la indexación
- Los agentes subsiguientes del mismo codebase reciben el índice ya existente
- El índice se libera cuando el último agente de ese codebase se desregistra (configurable: TTL de retención)

```go
type IndexManager struct {
    mu      sync.RWMutex
    indexes map[string]*CodebaseIndex  // key: codebase path (canonicalizado)
    refs    map[string]int             // conteo de agentes por codebase
}
```

---

## Pruebas de contrato multitenant

Nuevos tests a añadir en `tests/`:

```bash
# tests/test_multitenant_contract.sh

# 1. Registrar N agentes en el mismo codebase → solo 1 proceso de indexación
# 2. Verificar que todos los agentes reciben resultados consistentes
# 3. Desregistrar todos los agentes → índice se libera (o respeta TTL)
# 4. Registrar agentes en codebases distintos → índices independientes
# 5. Caída abrupta de cliente → daemon detecta y limpia la sesión
```

---

## Beneficios medibles

| Escenario | Antes | Después |
|-----------|-------|---------|
| 3 agentes, mismo codebase | 3× RAM del índice | 1× RAM del índice |
| Indexación inicial | 3 procesos paralelos | 1 proceso, 3 clientes esperan |
| Cambio de archivo | 3 re-indexaciones | 1 re-indexación compartida |
