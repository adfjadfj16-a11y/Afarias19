# Seguridad del SDK

## Tabla de tipos permitidos

| Tipo | Descripción |
| --- | --- |
| Permit2 `PermitWitnessTransferFrom` | Permit2 con witness - requiere opt-in explícito, validación bounded y límites de precio. No otorga permiso de pago nuevo. |

### Nested EIP-712 Signing (Bounded)

El checker de TypeScript soporta structs anidados EIP-712 cuando el grafo es bounded y existe un único primary type inferible, incluyendo arrays de structs (`TokenPermissions[]`, etc). La validación ocurre antes de firmar y cualquier violación lanza `PolicyViolation`.

**Límites de recurso (enforced):**
- Máx. 256 structs
- Máx. 4096 fields totales
- Profundidad máx. 64
- Identificadores máx. 256 caracteres
- Grafos ambiguos, desconectados, cíclicos, oversized o demasiado profundos son rechazados
- Regex anclado para mitigar ReDoS

**Compatibilidad:**
`inferPrimaryType` en TypeScript está actualmente por delante del port de Python. El port de Python todavía requiere un único struct no-domain.

> ⚠️ **Security Boundary de Permit2:** Agregar `PermitWitnessTransferFrom` al allowlist NO es suficiente. Las allowlists normales de primary-type/domain, la denylist y los validity checks siguen corriendo: nested signing no otorga un nuevo permiso de pago. **Nunca reenvíes raw typed data no confiable solo por haber agregado Permit2 al allowlist.** Valida por tu cuenta el witness, token (`0x...`), amount y spender con límites acotados (ej. `setPermit2Allowance(USDC, 50_000_000n)`).
