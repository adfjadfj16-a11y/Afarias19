#!/usr/bin/env python3
"""Testnet market-data / paper-trading bot. This program never submits orders."""

import argparse
import csv
import json
import math
import os
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

from private_ledger import (
    append_event,
    default_data_dir,
    ensure_outside_repository,
    import_binance_csv,
    secure_mkdir,
    verify_ledger,
)

TESTNET_BASE_URL = "https://testnet.binance.vision"
SYMBOL = os.getenv("SYMBOL", "PEPEUSDT").upper()
INTERVAL = os.getenv("INTERVAL", "5m")
INTERVAL_SECONDS = {
    "1m": 60,
    "3m": 180,
    "5m": 300,
    "15m": 900,
    "30m": 1800,
    "1h": 3600,
    "2h": 7200,
    "4h": 14400,
    "6h": 21600,
    "8h": 28800,
    "12h": 43200,
    "1d": 86400,
    "3d": 259200,
    "1w": 604800,
}
QUOTE_ORDER_SIZE = float(os.getenv("QUOTE_ORDER_SIZE", "5"))
FEE_RATE = float(os.getenv("PAPER_FEE_RATE", "0.001"))
FAST_EMA = 20
SLOW_EMA = 50
RSI_PERIOD = 14
ATR_PERIOD = 14
DATA_DIR = default_data_dir()
STATE_FILE = Path(os.getenv("STATE_FILE", DATA_DIR / "position_spot_testnet.json"))
CSV_FILE = Path(os.getenv("CSV_FILE", DATA_DIR / "trades_spot_testnet.csv"))
LEDGER_FILE = Path(os.getenv("LEDGER_FILE", DATA_DIR / "investment-ledger.jsonl"))


@dataclass(frozen=True)
class Candle:
    open_time: int
    open: float
    high: float
    low: float
    close: float
    close_time: int


@dataclass(frozen=True)
class Indicators:
    fast_ema: float
    slow_ema: float
    rsi: float
    atr: float


def fetch_closed_candles(symbol=SYMBOL, interval=INTERVAL, limit=100):
    query = urllib.parse.urlencode(
        {"symbol": symbol, "interval": interval, "limit": limit}
    )
    request = urllib.request.Request(
        f"{TESTNET_BASE_URL}/api/v3/klines?{query}",
        headers={"User-Agent": "safe-binance-paper-bot/1.0"},
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            rows = json.loads(response.read().decode("utf-8"))
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as error:
        raise RuntimeError(f"Could not fetch Testnet candles: {error}") from error

    if not isinstance(rows, list):
        raise RuntimeError("Unexpected candle response from Binance Testnet.")
    now_ms = int(time.time() * 1000)
    candles = []
    try:
        for row in rows:
            candle = Candle(
                open_time=int(row[0]),
                open=float(row[1]),
                high=float(row[2]),
                low=float(row[3]),
                close=float(row[4]),
                close_time=int(row[6]),
            )
            if not all(
                math.isfinite(value) and value > 0
                for value in (candle.open, candle.high, candle.low, candle.close)
            ):
                raise ValueError("Candle contains an invalid price.")
            if (
                candle.high < candle.low
                or not candle.low <= candle.open <= candle.high
                or not candle.low <= candle.close <= candle.high
            ):
                raise ValueError("Candle OHLC values are inconsistent.")
            if candle.close_time < now_ms:
                candles.append(candle)
    except (IndexError, TypeError, ValueError) as error:
        raise RuntimeError(f"Invalid candle data from Binance Testnet: {error}") from error

    if len(candles) < SLOW_EMA + 1:
        raise RuntimeError(
            f"Need at least {SLOW_EMA + 1} closed candles; received {len(candles)}."
        )
    if any(a.open_time >= b.open_time for a, b in zip(candles, candles[1:])):
        raise RuntimeError("Candle timestamps are not strictly increasing.")
    return candles


def ema(values, period):
    if period < 1 or len(values) < period:
        raise ValueError("Not enough values to calculate EMA.")
    result = sum(values[:period]) / period
    multiplier = 2 / (period + 1)
    for value in values[period:]:
        result = value * multiplier + result * (1 - multiplier)
    return result


def calculate_indicators(candles):
    closes = [candle.close for candle in candles]
    if len(closes) < max(SLOW_EMA, RSI_PERIOD + 1, ATR_PERIOD + 1):
        raise ValueError("Not enough candles to calculate the strategy indicators.")

    gains = []
    losses = []
    for previous, current in zip(closes[-RSI_PERIOD - 1 : -1], closes[-RSI_PERIOD:]):
        change = current - previous
        gains.append(max(change, 0))
        losses.append(max(-change, 0))
    average_gain = sum(gains) / RSI_PERIOD
    average_loss = sum(losses) / RSI_PERIOD
    if average_loss == 0:
        rsi = 100.0 if average_gain else 50.0
    else:
        rsi = 100 - 100 / (1 + average_gain / average_loss)

    true_ranges = []
    for previous, candle in zip(candles[-ATR_PERIOD - 1 : -1], candles[-ATR_PERIOD:]):
        true_ranges.append(
            max(
                candle.high - candle.low,
                abs(candle.high - previous.close),
                abs(candle.low - previous.close),
            )
        )
    return Indicators(
        fast_ema=ema(closes, FAST_EMA),
        slow_ema=ema(closes, SLOW_EMA),
        rsi=rsi,
        atr=sum(true_ranges) / ATR_PERIOD,
    )


def default_state():
    return {
        "version": 1,
        "symbol": SYMBOL,
        "interval": INTERVAL,
        "in_position": False,
        "quantity": 0.0,
        "entry_price": 0.0,
        "quote_spent": 0.0,
        "stop_price": 0.0,
        "take_profit_price": 0.0,
        "last_candle_time": 0,
    }


def load_state(path=STATE_FILE):
    try:
        with path.open(encoding="utf-8") as state_file:
            state = json.load(state_file)
    except FileNotFoundError:
        return default_state()
    if not isinstance(state, dict):
        raise RuntimeError(f"Invalid state file: {path}")
    if state.get("version") != 1:
        raise RuntimeError(f"Unsupported state version in {path}; preserve or rename it before restarting.")
    state = {**default_state(), **state}
    if state["symbol"] != SYMBOL or state["interval"] != INTERVAL:
        raise RuntimeError(
            f"State file {path} belongs to {state['symbol']} / {state['interval']}, "
            f"not {SYMBOL} / {INTERVAL}."
        )
    if not isinstance(state["in_position"], bool):
        raise RuntimeError(f"Invalid position flag in state file: {path}")
    if not isinstance(state["last_candle_time"], int) or state["last_candle_time"] < 0:
        raise RuntimeError(f"Invalid last_candle_time in state file: {path}")
    for key in (
        "quantity",
        "entry_price",
        "quote_spent",
        "stop_price",
        "take_profit_price",
    ):
        if not isinstance(state[key], (int, float)) or not math.isfinite(state[key]):
            raise RuntimeError(f"Invalid {key} in state file: {path}")
    if state["in_position"] and (
        state["quantity"] <= 0
        or state["entry_price"] <= 0
        or state["quote_spent"] <= 0
        or state["stop_price"] <= 0
        or state["take_profit_price"] <= state["stop_price"]
    ):
        raise RuntimeError(f"Inconsistent open position in state file: {path}")
    return state


def save_state(state, path=STATE_FILE):
    path = ensure_outside_repository(path)
    secure_mkdir(path.parent)
    temporary_path = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            dir=path.parent,
            prefix=f".{path.name}.",
            delete=False,
        ) as state_file:
            temporary_path = Path(state_file.name)
            json.dump(state, state_file, indent=2)
            state_file.write("\n")
        os.chmod(temporary_path, 0o600)
        temporary_path.replace(path)
    finally:
        if temporary_path and temporary_path.exists():
            temporary_path.unlink()


def append_trade(action, price, quantity, pnl=None, path=CSV_FILE):
    path = ensure_outside_repository(path)
    secure_mkdir(path.parent)
    is_new = not path.exists()
    with path.open("a", newline="", encoding="utf-8") as trade_file:
        if os.name == "posix":
            os.chmod(path, 0o600)
        writer = csv.writer(trade_file)
        if is_new:
            writer.writerow(
                ["timestamp_utc", "mode", "symbol", "action", "price", "quantity", "pnl_usdt"]
            )
        writer.writerow(
            [
                datetime.now(timezone.utc).isoformat(),
                "PAPER",
                SYMBOL,
                action,
                f"{price:.12g}",
                f"{quantity:.12g}",
                "" if pnl is None else f"{pnl:.12g}",
            ]
        )


def process_candle(state, candle, indicators):
    if candle.close_time <= state["last_candle_time"]:
        return state, None

    price = candle.close
    action = None
    if state["in_position"]:
        stop_hit = candle.low <= state["stop_price"]
        target_hit = candle.high >= state["take_profit_price"]
        if stop_hit or target_hit:
            # If both levels occur in one candle, assume the less favorable fill.
            price = (
                min(state["stop_price"], candle.open)
                if stop_hit
                else state["take_profit_price"]
            )
            action = "PAPER_SELL_STOP" if stop_hit else "PAPER_SELL_TARGET"
        elif indicators.fast_ema < indicators.slow_ema or indicators.rsi < 45:
            action = "PAPER_SELL_SIGNAL"
        if action:
            proceeds = state["quantity"] * price * (1 - FEE_RATE)
            pnl = proceeds - state["quote_spent"]
            new_state = default_state()
            new_state["last_candle_time"] = candle.close_time
            return new_state, {
                "action": action,
                "price": price,
                "quantity": state["quantity"],
                "pnl": pnl,
            }
    elif (
        indicators.fast_ema > indicators.slow_ema
        and 50 <= indicators.rsi <= 68
        and indicators.atr > 0
    ):
        quantity = QUOTE_ORDER_SIZE * (1 - FEE_RATE) / price
        state = {
            **state,
            "in_position": True,
            "quantity": quantity,
            "entry_price": price,
            "quote_spent": QUOTE_ORDER_SIZE,
            "stop_price": max(price - 2 * indicators.atr, price * 0.001),
            "take_profit_price": price + 3 * indicators.atr,
        }
        action = {
            "action": "PAPER_BUY",
            "price": price,
            "quantity": quantity,
            "pnl": None,
        }

    state = {**state, "last_candle_time": candle.close_time}
    return state, action


def run_once():
    if os.getenv("BINANCE_TESTNET") != "1":
        raise RuntimeError("Set BINANCE_TESTNET=1. Only Binance Testnet is supported.")
    if os.getenv("ENABLE_LIVE_TRADING", "NO").upper() != "NO":
        raise RuntimeError("Safety stop: ENABLE_LIVE_TRADING must remain NO.")
    if not SYMBOL.endswith("USDT") or not math.isfinite(QUOTE_ORDER_SIZE) or QUOTE_ORDER_SIZE <= 0:
        raise RuntimeError("Use a valid USDT-quoted symbol and a positive quote order size.")
    if INTERVAL not in INTERVAL_SECONDS:
        raise RuntimeError(f"Unsupported candle interval: {INTERVAL}")
    if not math.isfinite(FEE_RATE) or not 0 <= FEE_RATE < 0.05:
        raise RuntimeError("PAPER_FEE_RATE must be between 0 and 0.05.")

    candles = fetch_closed_candles()
    indicators = calculate_indicators(candles)
    state = load_state()
    candle = candles[-1]
    if candle.close_time <= state["last_candle_time"]:
        print("Closed candle already recorded; no duplicate evaluation.")
        return
    new_state, trade = process_candle(state, candles[-1], indicators)
    append_event(
        LEDGER_FILE,
        "paper_evaluation",
        {
            "event_id": f"paper:{SYMBOL}:{INTERVAL}:{candle.close_time}",
            "symbol": SYMBOL,
            "interval": INTERVAL,
            "candle_close_time": candle.close_time,
            "close": candle.close,
            "fast_ema": indicators.fast_ema,
            "slow_ema": indicators.slow_ema,
            "rsi": indicators.rsi,
            "atr": indicators.atr,
            "action": trade["action"] if trade else "HOLD",
            "paper_fill": trade,
        },
    )
    save_state(new_state)
    print(
        f"{SYMBOL} {INTERVAL} close={candle.close:.12g} "
        f"EMA{FAST_EMA}={indicators.fast_ema:.12g} "
        f"EMA{SLOW_EMA}={indicators.slow_ema:.12g} "
        f"RSI{RSI_PERIOD}={indicators.rsi:.2f} ATR{ATR_PERIOD}={indicators.atr:.12g}"
    )
    if trade:
        append_trade(**trade)
        print(f"{trade['action']} simulated; no order was sent.")
    else:
        print("No trade. No order was sent.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--loop", action="store_true", help="Check once per candle interval.")
    commands = parser.add_mutually_exclusive_group()
    commands.add_argument("--verify-ledger", action="store_true", help="Verify local ledger integrity and exit.")
    parser.add_argument(
        "--import-binance-csv",
        metavar="PATH",
        help="Import a manually exported Binance trade-history CSV into the separate local ledger.",
    )
    commands.add_argument(
        "--backtest",
        metavar="OHLCV_CSV",
        help="Run a chronological paper backtest on a local OHLCV CSV.",
    )
    commands.add_argument(
        "--walk-forward",
        metavar="OHLCV_CSV",
        help="Evaluate train, chronological walk-forward folds, and a final untouched holdout.",
    )
    parser.add_argument("--folds", type=int, default=4, help="Number of walk-forward validation folds.")
    parser.add_argument("--initial-cash", type=float, default=1000.0)
    parser.add_argument("--spread-bps", type=float, default=2.0)
    parser.add_argument("--slippage-bps", type=float, default=2.0)
    args = parser.parse_args()
    if args.verify_ledger:
        print(json.dumps(verify_ledger(LEDGER_FILE), indent=2))
        return 0
    if args.import_binance_csv:
        result = import_binance_csv(args.import_binance_csv, LEDGER_FILE)
        print(json.dumps(result, indent=2))
        return 0
    if args.backtest or args.walk_forward:
        from binance_backtest import backtest, load_candles_csv, walk_forward

        try:
            candles = load_candles_csv(args.backtest or args.walk_forward)
            if INTERVAL not in INTERVAL_SECONDS:
                raise ValueError(f"Unsupported candle interval: {INTERVAL}")
            options = {
                "initial_cash": args.initial_cash,
                "quote_size": QUOTE_ORDER_SIZE,
                "commission": FEE_RATE,
                "spread_bps": args.spread_bps,
                "slippage_bps": args.slippage_bps,
                "interval": INTERVAL,
            }
            result = (
                backtest(candles, SLOW_EMA, len(candles), **options)
                if args.backtest
                else walk_forward(candles, folds=args.folds, **options)
            )
        except (OSError, ValueError) as error:
            print(f"Backtest stopped safely: {error}", file=sys.stderr)
            return 1
        print(json.dumps(result, indent=2, allow_nan=False))
        return 0
    while True:
        try:
            run_once()
        except (RuntimeError, OSError, ValueError) as error:
            print(f"Stopped safely: {error}", file=sys.stderr)
            return 1
        if not args.loop:
            return 0
        time.sleep(INTERVAL_SECONDS[INTERVAL])


if __name__ == "__main__":
    raise SystemExit(main())
