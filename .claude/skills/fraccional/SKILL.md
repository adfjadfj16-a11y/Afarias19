---
name: fraccional
description: Operate your Fraccional (fraccional.cl) real-estate investment account from the terminal — log in with an email code, view your portfolio, movements, valuations, rent payrolls and the secondary market, and run read queries against the Fraccional API with curl. Use when the user mentions Fraccional, fraccional.cl, or their investments / fractions / portfolio / "mis inversiones" / "mi portafolio" there.
metadata:
  author: urvana
  version: "0.1.2"
---

# Fraccional desde la terminal

Opera tu cuenta de **[Fraccional](https://www.fraccional.cl)** (inversión inmobiliaria
fraccionada) con `curl`. Login por código al email, sesión persistente con auto-refresh, y
consultas REST a la API (Supabase PostgREST). Todo lo tuyo, protegido por RLS por usuario.

> Esto opera **tu propia cuenta**. La sesión guardada da acceso total a ella. Cierra sesión
> con el bloque *Logout* cuando termines en una máquina compartida.

## Requisitos

- `curl` y `jq` instalados (`brew install jq` en macOS).
- Una cuenta en https://www.fraccional.cl (cualquier método: email o Google/Facebook/LinkedIn).

## Constantes (públicas)

Todos los bloques empiezan con estas 3 líneas. `KEY` es la *publishable key* (pública, ya viaja
en el sitio web — **no** es un secreto). La sesión sí es secreta y va en `SESSION`.

```bash
SUPABASE_URL="https://api.fraccional.app"
KEY="sb_publishable_zsc5qocAVfPLJnD9IOHrpw_mtUlCg-7"
SESSION="$HOME/.fraccional/session.json"
```

## Reglas de seguridad (para el agente)

- **NUNCA** imprimas `access_token` ni `refresh_token` en pantalla (van al transcript). Los bloques
  los canalizan directo a `$SESSION` con `jq`.
- **NUNCA** uses `curl -v` ni `-i` en llamadas autenticadas (imprimen el header `Authorization`).
- **NUNCA** imprimas la respuesta cruda de `/auth/v1/otp`, `/auth/v1/verify` o `/auth/v1/token`
  (traen tokens). Solo extrae con `jq` lo no sensible (estado, email).
- El archivo `$SESSION` se crea con `chmod 600`.
- Las **mutaciones** (escribir/cancelar/comprar) se confirman con el usuario **antes** de ejecutar.

---

## Iniciar sesión

Dos formas; **ambas terminan en el _Paso final_** (verificar). En las dos, el usuario obtiene un
**código de 6 dígitos** que pega aquí — nunca su contraseña.

- **A. Navegador (recomendado):** abre una página, inicia sesión como siempre (email o
  Google/Facebook/LinkedIn) y copia el código. Nada de su email/clave pasa por el agente.
- **B. Código al email (curl puro):** pide el código de 6 dígitos al email de la cuenta.

### Opción A — navegador (`/app/cli`)

1. Abre la página de conexión:

   ```bash
   open "https://www.fraccional.cl/app/cli" 2>/dev/null \
     || echo "Abre https://www.fraccional.cl/app/cli en tu navegador"
   ```

2. Dile al usuario: inicia sesión (o ya lo estás) y pulsa **Generar código para la CLI**. La
   página muestra un **código de 6 dígitos** y tu email.
3. Pregúntale ese **código** y su **email**, y ve al **Paso final**. Crea una sesión
   **independiente** (no cierra la sesión web).

### Opción B — código al email

Funciona para **todos** los usuarios (incluidos los de Google/Facebook/LinkedIn): el código
llega al email de la cuenta. Pregunta el email al usuario y envíalo (`create_user:false` evita
crear cuentas nuevas):

```bash
SUPABASE_URL="https://api.fraccional.app"
KEY="sb_publishable_zsc5qocAVfPLJnD9IOHrpw_mtUlCg-7"
EMAIL="tucorreo@ejemplo.com"   # <-- pídeselo al usuario

curl -sS -X POST "$SUPABASE_URL/auth/v1/otp" \
  -H "apikey: $KEY" -H "content-type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"create_user\":false}" \
  | jq -r 'if .msg then "ERROR: " + .msg + (if .error_code then " (" + .error_code + ")" else "" end)
           elif .error_description then "ERROR: " + .error_description
           else "✓ Código enviado a '"$EMAIL"'. Revisa tu correo." end'
```

El correo trae un **código de 6 dígitos** (y un enlace de respaldo). Ve al **Paso final**.

### Paso final — verificar y guardar la sesión

Pide al usuario el **código de 6 dígitos** (de la página `/app/cli` o del correo) y su **email**.
Este bloque también acepta el enlace mágico completo: prueba las variantes de `type` hasta que una
funcione.

```bash
SUPABASE_URL="https://api.fraccional.app"
KEY="sb_publishable_zsc5qocAVfPLJnD9IOHrpw_mtUlCg-7"
SESSION="$HOME/.fraccional/session.json"
EMAIL="tucorreo@ejemplo.com"
INPUT="123456"   # <-- código de 6 dígitos, O el enlace completo (correo / página /cli)

mkdir -p "$(dirname "$SESSION")"

# Prueba un cuerpo de verificación; si trae access_token lo guarda y retorna 0.
verify_try() {
  local resp; resp=$(curl -sS -X POST "$SUPABASE_URL/auth/v1/verify" \
    -H "apikey: $KEY" -H "content-type: application/json" -d "$1")
  if printf '%s' "$resp" | jq -e 'has("access_token") and .access_token != null' >/dev/null 2>&1; then
    printf '%s' "$resp" | jq '{access_token, refresh_token, expires_at, email: .user.email}' > "$SESSION"
    chmod 600 "$SESSION"; return 0
  fi
  return 1
}

if printf '%s' "$INPUT" | grep -q '://'; then
  HASH=$(printf '%s' "$INPUT" | sed -n 's/.*[?&]token[=]\([^&]*\).*/\1/p')
  verify_try "{\"type\":\"magiclink\",\"token_hash\":\"$HASH\"}" \
    || verify_try "{\"type\":\"email\",\"token_hash\":\"$HASH\"}"
else
  verify_try "{\"type\":\"email\",\"email\":\"$EMAIL\",\"token\":\"$INPUT\"}" \
    || verify_try "{\"type\":\"magiclink\",\"email\":\"$EMAIL\",\"token\":\"$INPUT\"}"
fi

if [ -n "$(jq -r '.access_token // empty' "$SESSION" 2>/dev/null)" ]; then
  echo "✓ Sesión iniciada como $(jq -r '.email' "$SESSION")"
else
  rm -f "$SESSION"
  echo "✗ No se pudo verificar. Revisa el código/enlace o pide uno nuevo (Opción A o B)."
fi
unset INPUT HASH
```

---

## Antes de cada llamada: refrescar token

El `access_token` dura ~1h. Corre este bloque **antes** de cualquier query: refresca si falta <60s
y **persiste el nuevo refresh_token** (Supabase lo rota en cada uso).

```bash
SUPABASE_URL="https://api.fraccional.app"
KEY="sb_publishable_zsc5qocAVfPLJnD9IOHrpw_mtUlCg-7"
SESSION="$HOME/.fraccional/session.json"

[ -f "$SESSION" ] || { echo "No hay sesión. Corre el login primero."; exit 1; }
EXP=$(jq -r '.expires_at // 0' "$SESSION"); NOW=$(date +%s)
if [ "$NOW" -ge "$((EXP - 60))" ]; then
  RT=$(jq -r '.refresh_token' "$SESSION")
  resp=$(curl -sS -X POST "$SUPABASE_URL/auth/v1/token?grant_type=refresh_token" \
    -H "apikey: $KEY" -H "content-type: application/json" \
    -d "{\"refresh_token\":\"$RT\"}")
  if [ -n "$(printf '%s' "$resp" | jq -r '.access_token // empty')" ]; then
    printf '%s' "$resp" | jq '{access_token, refresh_token, expires_at, email: .user.email}' > "$SESSION.tmp" \
      && mv "$SESSION.tmp" "$SESSION" && chmod 600 "$SESSION"
    echo "✓ token refrescado"
  else
    echo "✗ refresh falló — vuelve a iniciar sesión"
  fi
  unset resp RT
else
  echo "✓ token vigente"
fi
```

## Correr una query (genérico)

Patrón base: lee el token del archivo, llama a PostgREST, muestra JSON. Para recursos scopeados al
usuario, **prefiere `viewer_*` y funciones curadas** en vez de tablas raw. (Corre primero el bloque
de refrescar.)

```bash
SUPABASE_URL="https://api.fraccional.app"
KEY="sb_publishable_zsc5qocAVfPLJnD9IOHrpw_mtUlCg-7"
SESSION="$HOME/.fraccional/session.json"
TOKEN=$(jq -r '.access_token' "$SESSION")

curl -sS -X POST "$SUPABASE_URL/rest/v1/rpc/viewer_profile" \
  -H "apikey: $KEY" -H "Authorization: Bearer $TOKEN" \
  -H "content-type: application/json" -d '{}' | jq
unset TOKEN
```

## Logout

```bash
rm -f "$HOME/.fraccional/session.json" && echo "✓ sesión cerrada"
```

---

## Catálogo de queries típicas (PostgREST `/rest/v1`)

La RLS scopea automáticamente "lo tuyo", pero para recursos del usuario **prefiere funciones
`viewer_*`** en vez de tablas raw. Reemplaza el `?path?query` del bloque genérico por:

| Intención | Endpoint |
|---|---|
| Mi perfil | `rpc/viewer_profile` *(POST, body `{}`)* |
| Mis movimientos / PnL | `rpc/viewer_purchase_confirmations_pnls?select=purchase_confirmation_id,unit_id,confirmed_at,status,bid_token_quantity,bid_preferred_amount,bid_preferred_currency,original_investment,current_value,pnl_amount,pnl_percentage&order=confirmed_at.desc&limit=20` *(POST, body `{}` o con `{"indicator_id":"CLP"}`)* |
| Resumen portafolio por moneda | `profile_portfolio_summary_by_currency?select=profile_id,valuation_currency,currency_amount,amount_clp` |
| **% propiedad en una unidad** | `purchase_confirmations_portfolios_pct?select=profile_id,unit_id,bid_token_quantity,pct&profile_id=eq.<PROFILE_ID>` *(filtra por `unit_id=eq.<ID>` para una sola; `pct` = fracción del total de tokens)* |
| Mis cuentas bancarias | `rpc/viewer_profile_bank_accounts?select=*` *(POST, body `{}`)* |
| Mis ventas activas (asks) | `rpc/viewer_profile_asks?select=profile_ask_id,unit_id,token_price,currency,ask_public_serial,created_at,cancelled_at&order=created_at.desc` *(POST, body `{}`)* |
| Mis arriendos cobrados | `rpc/viewer_payrolls?select=payroll_year,payroll_month,payroll_unit_id,transaction_amount,indicator_id,transaction_timestamped_at,payroll_reinvest_toggled&order=transaction_timestamped_at.desc&limit=20` *(POST, body `{}`; requiere headers adicionales; `indicator_id` = moneda)* |
| Mis retiros | `rpc/viewer_profile_withdrawals?select=profile_withdrawal_id,withdrawal_amount,withdrawal_currency,withdrawal_at,profile_withdrawal_type,unit_id&order=withdrawal_at.desc&limit=20` *(POST, body `{}`)* |
| Mis depósitos | `rpc/viewer_profile_charges?select=id,result_amount,result_currency,result_status,created_at&order=created_at.desc&limit=20` *(POST, body `{}`)* |
| Mercado: asks más baratos de una unidad | `secondary_market_orders?select=order_id,unit_id,order_type,order_amount,order_token_quantity,order_remaining_budget,currency,token_price,created_at&order_type=eq.ask&unit_id=eq.<ID>&order=token_price.asc&limit=5` *(nota: `token_price` = precio por fracción; `order_amount` = monto total de la orden)* |
| Mercado: mejores bids de una unidad | `secondary_market_orders?select=order_id,unit_id,order_type,order_amount,order_token_quantity,order_remaining_budget,currency,token_price,created_at&order_type=eq.bid&unit_id=eq.<ID>&order=token_price.desc&limit=5` |
| Mejores ofertas (oportunidades) | `rpc/profile_asks_summary_grouped_as?select=unit_id,currency,asks_count,token_quantity,asked_amount,reference_diff_pct,first_ask_public_serial&reference_diff_pct=lt.0&order=reference_diff_pct.asc&limit=20` *(POST, body `{"indicator_id":"CLP"}`)* |
| Propiedades / unidades | `units?select=id,name,slug,published_at,is_sold_out,rent_state&disabled=eq.false&published_at=not.is.null&order=published_at.desc&limit=20` |
| Detalle de una unidad | `units?id=eq.<ID>&disabled=eq.false&published_at=not.is.null&select=*` |
| Arriendos de una unidad | `unit_rentals?unit_id=eq.<ID>&select=id,unit_id,year_number,month_number,rent_nonce,rent_current_amount,rent_current_currency,rent_expected_amount,rent_expected_currency,rent_payment_date,confirmed_at&order=year_number.desc,month_number.desc,rent_nonce.desc` |
| **Valor actual de una unidad** | `unit_valuations?unit_id=eq.<ID>&select=id,unit_id,valuation_amount,valuation_currency,valuation_at,valuation_irr_expected_pct&order=valuation_at.desc&limit=1` *(primera fila = valuación vigente; usar siempre en vez de `funding_amount`/`funding_currency`)* |
| Historial de valuaciones | `unit_valuations?unit_id=eq.<ID>&select=id,unit_id,valuation_amount,valuation_currency,valuation_at,valuation_irr_expected_pct&order=valuation_at.desc` |

> `units` usa PK **`id`**; las tablas relacionadas la referencian como **`unit_id`**.
> En `units`, incluye siempre `published_at=not.is.null`.
> Para recursos del usuario, evita `profiles`, `profile_bank_accounts`, `profile_asks`,
> `purchase_confirmations_pnls`, `profile_withdrawals` y `profile_charges` directos.
> **Valor de una unidad** → `unit_valuations` con `order=valuation_at.desc&limit=1`. Nunca usar `funding_amount` ni `funding_currency` directamente.
> **% de propiedad** de un usuario en una unidad → vista `purchase_confirmations_portfolios_pct`;
> campo `pct` = `bid_token_quantity / tokens_disponibles`. Filtra `profile_id=eq.<ID>` y opcionalmente `unit_id=eq.<ID>`.

### Headers requeridos para algunas funciones

Algunos RPC (`viewer_payrolls`, y posiblemente otros) **requieren headers adicionales** además de `apikey` y `Authorization`:

```bash
curl -sS -X POST "$SUPABASE_URL/rest/v1/rpc/viewer_payrolls?select=*&order=transaction_timestamped_at.desc&limit=20" \
  -H "apikey: $KEY" \
  -H "Authorization: Bearer $TOKEN" \
  -H "content-type: application/json" \
  -H "content-profile: public" \
  -H "prefer: return=representation" \
  -H "x-client-info: supabase-js/2.4.0" \
  -d '{}' | jq
```

**Sin estos headers, la función retorna un array vacío `[]` sin error**, lo que silencia fallos RLS. Siempre incluye:

- `content-profile: public` — especifica el schema público.
- `prefer: return=representation` — pide la representación completa de lo que se retorna.
- `x-client-info: supabase-js/2.4.0` — identifica el cliente Supabase.

### RPC curados (POST)

Algunos atajos de la app son funciones. Se llaman con POST y cuerpo `{}` (o args). Úsalas como
primera opción cuando exista `viewer_*`:

```bash
SUPABASE_URL="https://api.fraccional.app"
KEY="sb_publishable_zsc5qocAVfPLJnD9IOHrpw_mtUlCg-7"
SESSION="$HOME/.fraccional/session.json"
TOKEN=$(jq -r '.access_token' "$SESSION")

curl -sS -X POST "$SUPABASE_URL/rest/v1/rpc/viewer_profile" \
  -H "apikey: $KEY" -H "Authorization: Bearer $TOKEN" \
  -H "content-type: application/json" -d '{}' | jq
unset TOKEN
```

Con argumentos:

```bash
SUPABASE_URL="https://api.fraccional.app"
KEY="sb_publishable_zsc5qocAVfPLJnD9IOHrpw_mtUlCg-7"
SESSION="$HOME/.fraccional/session.json"
TOKEN=$(jq -r '.access_token' "$SESSION")

curl -sS -X POST \
  "$SUPABASE_URL/rest/v1/rpc/profile_asks_summary_grouped_as?reference_diff_pct=lt.0&order=reference_diff_pct.asc&limit=20" \
  -H "apikey: $KEY" -H "Authorization: Bearer $TOKEN" \
  -H "content-type: application/json" \
  -d '{"indicator_id":"CLP"}' | jq
unset TOKEN
```

## Cheatsheet PostgREST

- **Filtros**: `?columna=eq.valor` (`eq, neq, gt, gte, lt, lte, like, ilike, in, is`).
  Ej: `?status=in.(active,pending)`, `?name=ilike.*centro*`.
- **Columnas**: `?select=id,name,funding_amount`.
- **Orden**: `?order=created_at.desc` (encadena: `?order=a.desc,b.asc`).
- **Paginación**: `?limit=20&offset=40`.
- **Embedding (joins)**: `?select=*,unit_valuations(*)` (tabla relacionada anidada).
- **Conteo exacto**: agrega header `-H "Prefer: count=exact"` y mira `Content-Range` en la respuesta.
- **Schema**: solo `public` (default). No se necesita header extra.

## Descubrir más

- ¿No conoces las columnas? Pide 1 fila: `...?select=*&limit=1` y mira las claves con `jq '.[0]|keys'`.
- ¿Buscas otra tabla/función? La RLS te limita a `public`. Prueba el nombre; si no existe,
  PostgREST devuelve `404` o sugiere el correcto en `hint`.
- Existe también GraphQL en `POST /rest/v1/graphql_extended` y `/graphql/v1` (queries raw, misma
  RLS). REST cubre la mayoría de los casos y es más compacto.

## Mutaciones (con confirmación)

PostgREST también escribe. **Confirma con el usuario antes de ejecutar** cualquiera de estas.

- Insertar fila: `POST /rest/v1/<tabla>` con header `-H "Prefer: return=representation"` y cuerpo JSON.
- Actualizar: `PATCH /rest/v1/<tabla>?<filtro>` con cuerpo JSON.
- Funciones de la app (curadas): `POST /rest/v1/rpc/<fn>`, p.ej.
  `rpc/viewer_profile_ask_cancel` (cancelar una venta), `rpc/viewer_activate_promo_code`
  (activar código). Pasa los argumentos en el cuerpo JSON.
- Re-inversiones: no hagas update in-place; inserta una nueva fila en `profile_settings_reinvests`
  porque es una serie temporal. El estado vigente se lee con `rpc/viewer_profile_settings_reinvests`.

## Troubleshooting

| Síntoma | Causa / Fix |
|---|---|
| `JWT expired` / `PGRST301` | Corre el bloque *refrescar token*. Si persiste, vuelve a iniciar sesión. |
| `401` en todas las queries | Falta `Authorization: Bearer` o sesión inválida → re-login. |
| `column X does not exist` | Mira el `hint`; usa `select=*&limit=1` para ver columnas reales. |
| `404` en una tabla | No existe o no está expuesta en `public`. Verifica el nombre. |
| Respuesta vacía `[]` | RLS: no tienes filas para ese recurso (esperado si no es tuyo). |
| `Código enviado` pero no llega | Revisa spam; el remitente es Fraccional. Reintenta la *Opción B*. |
| `Signups not allowed for otp` | El email no está registrado en Fraccional, o el OTP por email está desactivado. Verifica el correo o usa la *Opción A* (`/app/cli`). |
| `jq: command not found` | `brew install jq`. |
