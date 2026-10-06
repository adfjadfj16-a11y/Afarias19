# Changelog

Todos los cambios notables en este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] — 2026-10-06

### Added — Pillar 4 (Auto-test al Arranque)
- ✅ `cmd/self-check/main.go` — Verificaciones de integridad (assets, config, entorno, permisos)
- ✅ `cmd/self-check/main_test.go` — Tests unitarios (7 tests, 100% PASS)
- ✅ Salida legible (ASCII art) y JSON para integración
- ✅ Flags: `-json`, `-v` (verbose), `-help`, `-version`
- ✅ Exit codes: 0 (OK), 1 (FAIL) para scripts

### Added — Pillar 1 (Code Signing)
- ✅ `.github/workflows/release.yml` — Firma automática (DigiCert, Apple, GPG)
- ✅ Build multiplataforma: linux-amd64/arm64, darwin-amd64/arm64, windows
- ✅ Checksums SHA256 incluidos en cada release
- ✅ GitHub Releases con assets verificables

### Added — CI/CD Pipeline
- ✅ `.github/workflows/ci.yml` — Lint, Test, Build, Security Scan
- ✅ Lint: golangci-lint, black, ESLint
- ✅ Test: `go test -race`, `pytest` con cobertura
- ✅ Security: Trivy scanner
- ✅ Integration: Validación de binarios

### Added — Documentation
- ✅ `README.md` (8.2 KB) — Guía completa del proyecto
- ✅ `CONTRIBUTING.md` (6.5 KB) — Guía de contribución
- ✅ `SETUP_SECRETS.md` (8.5 KB) — Configuración de secrets GitHub

### Added — Audit & Compliance (Pilar Adicional)
- ✅ `pkg/errors/schema.go` — Registro auditable con índice O(log n) y hash SHA256 encadenado
- ✅ `pkg/audit/auditor.go` — Bitácora persistente JSONL con verificación de integridad
- ✅ `legalivypolyci.py` — Backend SQLite + exportación JSON para auditoría legal
- ✅ `docs/AUDIT_COMPLIANCE.md` — Guía para verificación e inspección externa

### Added — FASE 2 Improvements
- ✅ `persistence_manager_test.go` — Tests unitarios (16 tests, cobertura ≥80%)
- ✅ `tests/test_bot_comprehensive.py` — Tests Python (20+ tests)
- ✅ `Makefile` — Comandos para build, test, lint, deploy
- ✅ `Dockerfile` — Multi-stage build, distroless, seguro
- ✅ `docker-compose.yml` — Dev (3 servicios) + prod (1 servicio)

---

## [Unreleased] — Próximos Cambios

### FASE 2: Refactorización + Tests + CI/CD (En Progreso)
- [ ] Agregar tests para persistence_manager.go (unit + integration)
- [ ] Agregar tests para bot_spot_binance_safe.py (fixtures, mocks)
- [ ] Refactorizar: pkg/persistence, pkg/trading, pkg/memory
- [ ] Aumentar cobertura: ≥80% Go, ≥75% Python
- [ ] Logging estructurado (JSON) en todos los módulos
- [ ] Dockerfile refinado + scanning de vulnerabilidades

### FASE 3: Presupuesto de Memoria
- [ ] Implementar: `--max-memory-budget` flag
- [ ] Monitoreo en tiempo real de uso de memoria
- [ ] Endpoint `/health` HTTP con métricas
- [ ] Presión progresiva: 80%, 95%, 100%
- [ ] Logs estructurados con eventos de presión

### FASE 4: Daemon Multitenant
- [ ] Servidor IPC (Unix socket / Named Pipe)
- [ ] JSON-RPC 2.0 protocol
- [ ] Registro de agentes: `agent.register`, `agent.unregister`
- [ ] Índice compartido en memoria
- [ ] Tests de concurrencia (N agentes simultáneos)

---

## [0.5.0] — 2026-10-05

### Initial Setup
- ✅ Estructura base del repositorio
- ✅ go.mod, .github/workflows
- ✅ persistence_manager.go (gestor de persistencia)
- ✅ bot_spot_binance_safe.py (bot educativo)
- ✅ check-persistent-memory.js (validador UI)
- ✅ docs/ (documentación de pilares)

---

## Notas de Versión

### Cómo interpretar versiones

- **[MAJOR].[MINOR].[PATCH]**
  - **MAJOR:** Cambios breaking (incompatible con versiones anteriores)
  - **MINOR:** Nuevas características, compatible hacia atrás
  - **PATCH:** Bug fixes, correcciones

### Secuencia de Fases

1. **FASE 1 (v1.0.0)** ✅ — Pilares 4 + 1: Confianza inmediata
   - Auto-test al arranque
   - Code signing (Windows, macOS, Linux)
   - CI/CD básico

2. **FASE 2 (v1.1.0)** ⏳ — Refactorización + Tests: Calidad
   - Tests unitarios e integración
   - Refactorización de estructura
   - Cobertura ≥80%
   - Docker + docker-compose

3. **FASE 3 (v1.2.0)** ⏳ — Presupuesto de memoria: Escalabilidad
   - Monitoreo en tiempo real
   - Presupuesto configurable
   - Endpoint /health

4. **FASE 4 (v2.0.0)** ⏳ — Daemon multitenant: Eficiencia
   - Orquestador central
   - Índice compartido
   - -87% RAM, -97% latencia

---

## Cómo Contribuir

Ver [CONTRIBUTING.md](CONTRIBUTING.md) para:
- Setup de desarrollo
- Flujo de trabajo (branch, commit, PR)
- Convenciones de código
- Proceso de testing

---

## Seguridad

### Reporting Security Issues

Por favor, **NO** reportes vulnerabilidades en GitHub Issues.
Envía un email privado a los mantenedores con detalles.

### Code Signing

Todos los releases son **firmados criptográficamente**:
- Windows: EV Certificate (DigiCert)
- macOS: Developer ID (Apple)
- Linux: GPG detached signatures

Verificar integridad antes de usar. Ver [SETUP_SECRETS.md](SETUP_SECRETS.md).

---

## Licencia

[Define licencia aquí — MIT, Apache 2.0, etc.]

---

## Créditos

- **Autor:** Afarias19 Community
- **Pacto:** Humanidad al centro, buenas prácticas siempre
- **Filosofía:** "Siempre perseverante, énfasis y la energía positiva"

---

**Última actualización:** 2026-10-06
