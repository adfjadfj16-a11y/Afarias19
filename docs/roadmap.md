# Roadmap post-PR #1508: De herramienta estable a estándar corporativo

## Objetivo

Pasar de una herramienta que "funciona a pesar de los antivirus" a una infraestructura que "es un estándar corporativo de confianza".

## Pilares

| # | Pilar | Prioridad | Estado | Fecha objetivo |
|---|-------|-----------|--------|----------------|
| 0 | [Hotfix: cbm-integrations.json en ZIP Windows](./bugfix-issue-1509-cbm-integrations-missing.md) | **Crítica** | 🔴 En progreso | Próximo patch release |
| 1 | [Reputational Hardening (Code Signing)](./pillar-1-code-signing.md) | Alta | ⬜ Pendiente | Q4 2026 |
| 2 | [Consolidación del Daemon (Multitenant)](./pillar-2-multitenant-daemon.md) | Media | ⬜ Pendiente | Q1 2027 |
| 3 | [Observabilidad y Presupuesto de Memoria](./pillar-3-memory-budget.md) | Media | ⬜ Pendiente | Q1 2027 |
| 4 | [Integridad Verificable (Auto-test)](./pillar-4-self-check.md) | Alta | ⬜ Pendiente | Q4 2026 |

## Orden de ejecución

1. **Hotfix — cbm-integrations.json en ZIP Windows** — Bloqueante para usuarios Windows en v0.9.0. Parche inmediato disponible (Opción B). Ver detalles en [bugfix-issue-1509-cbm-integrations-missing.md](./bugfix-issue-1509-cbm-integrations-missing.md).
2. **Pilar 1 — Code Signing** — Impacto inmediato en confianza, elimina permanentemente las detecciones AV.
3. **Pilar 4 — Auto-test al arranque** — Bajo costo de implementación, alto valor para adopción corporativa.
4. **Pilar 3 — Presupuesto de memoria** — Crítico para entornos empresariales con múltiples agentes.
5. **Pilar 2 — Daemon multitenant** — Mayor complejidad arquitectural, requiere planificación cuidadosa.

## Contexto

Este roadmap surge tras la estabilización lograda en el PR #1508 (reversión de activos, correcciones arm64, manejo de falsos positivos AV). El PR #1508 fue una maniobra de reparación de daños necesaria; este roadmap es la visión a largo plazo.
