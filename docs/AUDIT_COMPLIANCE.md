# Audit & Compliance

## Esquema
- `pkg/errors/schema.go`: registro en memoria con índice por `code` y cadena hash SHA256.
- `pkg/audit/auditor.go`: bitácora persistente JSONL con `hash256_previous` + `hash256_current`.
- `legalivypolyci.py`: registro SQLite para auditoría legal y exportación JSON.

## Hash y cadena
- `hash256 = SHA256(code + message + timestamp + prev_hash)`.
- Cada evento referencia el hash anterior para detectar manipulación.
- Solo se almacenan `code`, `message` y metadatos públicos; nunca secretos.

## Ejemplos
### Go
```go
registry := errors.NewRegistry()
entry, _ := registry.Register(errors.ErrorSchema{
    Code: "ERR-CRITICAL",
    Message: "public compliance message",
    Severity: errors.SeverityCritical,
    Category: "self-check",
})
_ = entry
```

### Python
```python
from legalivypolyci import ErrorRegistry
registry = ErrorRegistry()
registry.register("ERR-CRITICAL", "public compliance message", {"service": "self-check"})
registry.export_json("audit/report.json")
```

## Verificación por auditor externo
1. Ejecutar `go test ./pkg/errors ./pkg/audit` y `pytest tests/test_audit_system.py`.
2. Validar la cadena persistente con `VerifyChain()` en Go o `verify_chain()` en Python.
3. Revisar el archivo JSON exportado y confirmar que cada `hash256` coincide con `code + message + timestamp + prev_hash`.
4. Confirmar que no existen secretos en `environment` y que solo hay mensajes públicos.
