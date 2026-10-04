# BNBAgent SDK - Security & Signing Policy

## Signing Policy por defecto

All Permit variants (ERC-2612 `Permit`, Permit2 `PermitSingle`/`PermitBatch`) are denylisted — incluso si tu código los agrega por error al allowlist, la denylist gana.

La amenaza: sin `SigningPolicy`, un agente LLM que recibe un challenge 402 de un servidor malicioso podría ser convencido de firmar un Permit con allowance ilimitado, drenando la wallet.

La política por defecto rechaza incondicionalmente; tú haces opt-in explícito solo cuando sabes lo que firmas.

## Tipos permitidos (con opt-in)

| Tipo | Notas |
| :--- | :--- |
| `EIP-712 Typed Data` genérico | Requiere allowlist de primary type + domain |
| Permit2 `PermitWitnessTransferFrom` | Permit2 con witness - requiere opt-in explícito, validación bounded y clamps de precio. No otorga un nuevo permiso de pago. |

## Nested EIP-712 Signing (Bounded) - Nuevo en SDK #89

El checker de TypeScript ahora soporta structs anidados cuando su grafo es bounded y tiene un único primary type inferible, incluyendo arrays de structs.

**Validación:**
- Ocurre ANTES de firmar
- Violaciones lanzan `PolicyViolation`
- Soporta `struct[]` (ej. `TokenPermissions[]` dentro de `PermitWitnessTransferFrom`)

**Límites de recurso:**
- Máx 256 structs
- Máx 4096 fields totales
- Profundidad máx 64
- Identificadores máx 256 chars
- Rechaza grafos ambiguos, desconectados, cíclicos, oversized
- Regex anclado para evitar ReDoS

**Compatibilidad:**
`inferPrimaryType` en TS está actualmente por delante del port de Python. El port de Python todavía requiere un único struct no-domain.

> ⚠️ **Security Boundary Crítico:**
> Agregar `PermitWitnessTransferFrom` al allowlist NO te da un permiso de pago nuevo. Las allowlists normales de primary-type/domain, la denylist y los validity checks siguen corriendo.
>
> **Nunca reenvíes raw typed data no confiable solo por haber agregado Permit2 al allowlist.**
>
> Debes validar por tu cuenta el witness, token, amount y spender con límites acotados. Ejemplo canónico:
>
> ```typescript
> await admin.approveX402SignatureChecker(session); // checker = Permit2
> await admin.setPermit2Allowance(USDC, 50_000_000n); // BOUNDED - ver abajo
> ```
