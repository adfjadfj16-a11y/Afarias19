# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es/1.0.0/).

---

## [Sin lanzar]

### Corregido
- Instalador de Windows (`install.ps1`) falla con v0.9.0 porque `cbm-integrations.json` no está incluido en el ZIP del release — ver [issue #1509](https://github.com/DeusData/codebase-memory-mcp/issues/1509) y [docs/bugfix-issue-1509-cbm-integrations-missing.md](docs/bugfix-issue-1509-cbm-integrations-missing.md)

### Pendiente
- Pilar 1: Firma de código (EV Windows + Notarización macOS + GPG Linux) — ver [docs/pillar-1-code-signing.md](docs/pillar-1-code-signing.md)
- Pilar 2: Daemon multitenant — ver [docs/pillar-2-multitenant-daemon.md](docs/pillar-2-multitenant-daemon.md)
- Pilar 3: Presupuesto de memoria y observabilidad — ver [docs/pillar-3-memory-budget.md](docs/pillar-3-memory-budget.md)
- Pilar 4: Auto-verificación al arranque (`--self-check`) — ver [docs/pillar-4-self-check.md](docs/pillar-4-self-check.md)

---

## [0.9.0] — 2026-08 (estimado)

### Agregado
- Soporte arm64 en Windows y macOS
- Manejo de falsos positivos de antivirus (reversión de heurísticas)
- Correcciones de instalación en múltiples plataformas (PR #1508 upstream)

### Conocido — Bug activo
- `install.ps1` en Windows no puede completar la instalación con el ZIP de v0.9.0 (falta `cbm-integrations.json`). **Plataformas afectadas:** Windows amd64 y arm64. **Workaround:** ninguno sin modificar el script manualmente.
