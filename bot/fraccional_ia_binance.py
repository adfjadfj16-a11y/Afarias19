"""
fraccional_ia_binance.py — Bot de trading fraccional con IA para Binance.

Estrategias soportadas:
  - sma_cross : cruce de medias móviles SMA 20 / SMA 50
  - rsi       : RSI < 30 compra, RSI > 70 vende
  - openai    : señal vía GPT usando datos de velas recientes
"""
import logging
import time

from binance.client import Client
from binance.exceptions import BinanceAPIException

from config import config

logging.basicConfig(
    level=getattr(logging, config.LOG_LEVEL.upper(), logging.INFO),
    format="%(asctime)s [%(levelname)s] %(message)s",
)
log = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Indicadores técnicos
# ---------------------------------------------------------------------------

def _closes(klines: list) -> list[float]:
    return [float(k[4]) for k in klines]


def sma(prices: list[float], period: int) -> float:
    if len(prices) < period:
        raise ValueError(f"Se necesitan al menos {period} precios para SMA-{period}.")
    return sum(prices[-period:]) / period


def rsi(prices: list[float], period: int = 14) -> float:
    if len(prices) < period + 1:
        raise ValueError(f"Se necesitan al menos {period + 1} precios para RSI-{period}.")
    deltas = [prices[i] - prices[i - 1] for i in range(1, len(prices))]
    gains = [d for d in deltas[-period:] if d > 0]
    losses = [-d for d in deltas[-period:] if d < 0]
    avg_gain = sum(gains) / period if gains else 0.0
    avg_loss = sum(losses) / period if losses else 0.0
    if avg_loss == 0:
        return 100.0
    rs = avg_gain / avg_loss
    return 100 - (100 / (1 + rs))


# ---------------------------------------------------------------------------
# Señales de trading
# ---------------------------------------------------------------------------

def signal_sma_cross(closes: list[float]) -> str:
    fast = sma(closes, 20)
    slow = sma(closes, 50)
    prev_fast = sma(closes[:-1], 20)
    prev_slow = sma(closes[:-1], 50)
    if prev_fast <= prev_slow and fast > slow:
        return "BUY"
    if prev_fast >= prev_slow and fast < slow:
        return "SELL"
    return "HOLD"


def signal_rsi(closes: list[float]) -> str:
    r = rsi(closes)
    log.debug("RSI actual: %.2f", r)
    if r < 30:
        return "BUY"
    if r > 70:
        return "SELL"
    return "HOLD"


def signal_openai(closes: list[float], pair: str) -> str:
    try:
        import openai  # type: ignore
        openai.api_key = config.OPENAI_API_KEY
        prompt = (
            f"Eres un experto en trading. Aquí están los últimos 20 precios de cierre "
            f"del par {pair}: {closes[-20:]}. "
            "Responde SOLO con una palabra: BUY, SELL o HOLD."
        )
        response = openai.chat.completions.create(
            model="gpt-4o-mini",
            messages=[{"role": "user", "content": prompt}],
            max_tokens=5,
            temperature=0,
        )
        signal = response.choices[0].message.content.strip().upper()
        if signal not in ("BUY", "SELL", "HOLD"):
            return "HOLD"
        return signal
    except Exception as exc:
        log.error("Error al consultar OpenAI: %s", exc)
        return "HOLD"


# ---------------------------------------------------------------------------
# Ejecución de órdenes fraccionales
# ---------------------------------------------------------------------------

def get_balance(client: Client, asset: str) -> float:
    info = client.get_asset_balance(asset=asset)
    return float(info["free"]) if info else 0.0


def place_order(client: Client, pair: str, side: str, fraction: float) -> None:
    quote_asset = pair[-4:] if pair.endswith("USDT") else pair[-3:]
    base_asset = pair[: -len(quote_asset)]

    if side == "BUY":
        balance = get_balance(client, quote_asset)
        spend = balance * fraction
        ticker = client.get_symbol_ticker(symbol=pair)
        price = float(ticker["price"])
        qty = spend / price
        info = client.get_symbol_info(pair)
        step = float(next(f["stepSize"] for f in info["filters"] if f["filterType"] == "LOT_SIZE"))
        qty = round(qty - (qty % step), 8)
        if qty <= 0:
            log.warning("Cantidad calculada ≤ 0, operación omitida.")
            return
        log.info("COMPRANDO %.8f %s a %.2f %s", qty, base_asset, price, quote_asset)
        client.order_market_buy(symbol=pair, quantity=qty)

    elif side == "SELL":
        balance = get_balance(client, base_asset)
        qty = balance * fraction
        info = client.get_symbol_info(pair)
        step = float(next(f["stepSize"] for f in info["filters"] if f["filterType"] == "LOT_SIZE"))
        qty = round(qty - (qty % step), 8)
        if qty <= 0:
            log.warning("Cantidad calculada ≤ 0, operación omitida.")
            return
        log.info("VENDIENDO %.8f %s", qty, base_asset)
        client.order_market_sell(symbol=pair, quantity=qty)


# ---------------------------------------------------------------------------
# Bucle principal
# ---------------------------------------------------------------------------

def run() -> None:
    config.validate()

    client = Client(
        api_key=config.BINANCE_API_KEY,
        api_secret=config.BINANCE_API_SECRET,
        testnet=config.BINANCE_TESTNET,
    )
    log.info(
        "Bot iniciado | par=%s | estrategia=%s | fracción=%.2f%% | testnet=%s",
        config.TRADING_PAIR,
        config.STRATEGY,
        config.TRADE_FRACTION * 100,
        config.BINANCE_TESTNET,
    )

    while True:
        try:
            klines = client.get_klines(
                symbol=config.TRADING_PAIR,
                interval=Client.KLINE_INTERVAL_1HOUR,
                limit=100,
            )
            closes = _closes(klines)

            if config.STRATEGY == "sma_cross":
                signal = signal_sma_cross(closes)
            elif config.STRATEGY == "rsi":
                signal = signal_rsi(closes)
            elif config.STRATEGY == "openai":
                signal = signal_openai(closes, config.TRADING_PAIR)
            else:
                log.error("Estrategia desconocida: %s", config.STRATEGY)
                signal = "HOLD"

            log.info("Señal: %s", signal)

            if signal in ("BUY", "SELL"):
                place_order(client, config.TRADING_PAIR, signal, config.TRADE_FRACTION)

        except BinanceAPIException as exc:
            log.error("Error de Binance API: %s", exc)
        except Exception as exc:
            log.exception("Error inesperado: %s", exc)

        log.debug("Esperando %d segundos...", config.POLL_INTERVAL_SECONDS)
        time.sleep(config.POLL_INTERVAL_SECONDS)


if __name__ == "__main__":
    run()
