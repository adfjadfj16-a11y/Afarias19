# Configuración de cuenta Binance para el Bot Fraccional IA

## 1. Crear cuenta en Binance

1. Ve a [https://www.binance.com](https://www.binance.com) y regístrate.
2. Completa la verificación de identidad (KYC) para habilitar el trading.

## 2. Generar API Keys

1. Inicia sesión → **Perfil** → **Gestión de API**.
2. Haz clic en **Crear API** → elige "API generada por el sistema".
3. Ponle un nombre descriptivo, ej: `fraccional-ia-bot`.
4. **Permisos recomendados (mínimos necesarios):**
   - ✅ Habilitar lectura
   - ✅ Habilitar trading al contado
   - ❌ NO habilitar retiros (por seguridad)
5. Guarda la **API Key** y el **API Secret** — el secret solo se muestra una vez.

> ⚠️ **Seguridad:** Restringe la IP a la del servidor donde corre el bot.

## 3. Testnet (recomendado para pruebas)

Usa la red de prueba de Binance para operar sin dinero real:

1. Entra a [https://testnet.binance.vision](https://testnet.binance.vision).
2. Genera API Keys de testnet.
3. En tu `.env`, pon `BINANCE_TESTNET=true`.

## 4. Configurar el `.env`

```bash
cp .env.example .env
```

Edita `.env` con tus credenciales:

```env
BINANCE_API_KEY=tu_api_key_aqui
BINANCE_API_SECRET=tu_api_secret_aqui
BINANCE_TESTNET=true      # Cambia a false cuando estés listo para real
TRADING_PAIR=BTCUSDT
TRADE_FRACTION=0.01       # 1% del saldo por operación
STRATEGY=sma_cross
```

## 5. Instalar dependencias

```bash
pip install -r requirements.txt
```

## 6. Ejecutar el bot

```bash
python bot/fraccional_ia_binance.py
```

## Estrategias disponibles

| Estrategia   | Descripción                                      |
|-------------|--------------------------------------------------|
| `sma_cross` | Cruce de medias móviles (SMA 20 / SMA 50)        |
| `rsi`       | RSI < 30 compra, RSI > 70 vende                  |
| `openai`    | Señal generada por GPT usando datos de mercado   |

> Para `openai`, necesitas también una `OPENAI_API_KEY` válida en tu `.env`.
