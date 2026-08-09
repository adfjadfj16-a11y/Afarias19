# Roadmap post-PR #1508: De herramienta estable a estándar corporativo

## Objetivo

Pasar de una herramienta que "funciona a pesar de los antivirus" a una infraestructura que "es un estándar corporativo de confianza".

## Pilares

| # | Pilar | Prioridad | Estado |
|---|-------|-----------|--------|
| 1 | [Reputational Hardening (Code Signing)](./pillar-1-code-signing.md) | Alta | Pendiente |
| 2 | [Consolidación del Daemon (Multitenant)](./pillar-2-multitenant-daemon.md) | Media | Pendiente |
| 3 | [Observabilidad y Presupuesto de Memoria](./pillar-3-memory-budget.md) | Media | Pendiente |
| 4 | [Integridad Verificable (Auto-test)](./pillar-4-self-check.md) | Alta | Pendiente |

## Orden de ejecución

1. **Pilar 1 — Code Signing** — Impacto inmediato en confianza, elimina permanentemente las detecciones AV.
2. **Pilar 4 — Auto-test al arranque** — Bajo costo de implementación, alto valor para adopción corporativa.
3. **Pilar 3 — Presupuesto de memoria** — Crítico para entornos empresariales con múltiples agentes.
4. **Pilar 2 — Daemon multitenant** — Mayor complejidad arquitectural, requiere planificación cuidadosa.

## Contexto

Este roadmap surge tras la estabilización lograda en el PR #1508 (reversión de activos, correcciones arm64, manejo de falsos positivos AV). El PR #1508 fue una maniobra de reparación de daños necesaria; este roadmap es la visión a largo plazo.
