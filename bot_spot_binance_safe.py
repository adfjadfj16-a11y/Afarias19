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
NETWORK_TIMEOUT = float(os.getenv("NETWORK_TIMEOUT", "5"))
MAX_RETRIES = int(os.getenv("MAX_RETRIES", "3"))
RETRY_DELAY = float(os.getenv("RETRY_DELAY", "1"))
FAST_EMA = 20
SLOW_EMA = 50
RSI_PERIOD = 14
ATR_PERIOD = 14
MIN_CANDLES_BUFFER = max(SLOW_EMA, RSI_PERIOD + 1, ATR_PERIOD + 1) + 1
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
    url = f"{TESTNET_BASE_URL}/api/v3/klines?{query}"
    request = urllib.request.Request(
        url,
        headers={"User-Agent": "safe-binance-paper-bot/1.0"},
    )
    
    last_error = None
    for attempt in range(1, MAX_RETRIES + 1):
        try:
            with urllib.request.urlopen(request, timeout=NETWORK_TIMEOUT) as response:
                rows = json.loads(response.read().decode("utf-8"))
                break
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as error:
            last_error = error
            if attempt < MAX_RETRIES:
                time.sleep(RETRY_DELAY * attempt)
                continue
            raise RuntimeError(f"Could not fetch Testnet candles after {MAX_RETRIES} attempts: {error}") from error

    if not isinstance(rows, list):
        raise RuntimeError("Unexpected candle response from Binance Testnet: not a list.")
    
    now_ms = int(time.time() * 1000)
    candles = []
    try:
        for row in rows:
            if not isinstance(row, (list, tuple)) or len(row) < 7:
                raise ValueError(f"Invalid candle row format: {row}")
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
                raise ValueError(f"Candle contains an invalid price: open={candle.open}, high={candle.high}, low={candle.low}, close={candle.close}")
            if (
                candle.high < candle.low
                or not candle.low <= candle.open <= candle.high
                or not candle.low <= candle.close <= candle.high
            ):
                raise ValueError(f"Candle OHLC values are inconsistent: {candle}")
            if candle.close_time < now_ms:
                candles.append(candle)
    except (IndexError, TypeError, ValueError) as error:
        raise RuntimeError(f"Invalid candle data from Binance Testnet: {error}") from error

    if len(candles) < MIN_CANDLES_BUFFER:
        raise RuntimeError(
            f"Need at least {MIN_CANDLES_BUFFER} closed candles; received {len(candles)}."
        )
    if any(a.open_time >= b.open_time for a, b in zip(candles, candles[1:])):
        raise RuntimeError("Candle timestamps are not strictly increasing.")
    return candles


def ema(values, period):
    if period < 1:
        raise ValueError(f"EMA period must be at least 1, got {period}.")
    if len(values) < period:
        raise ValueError(f"Not enough values to calculate EMA: need {period}, got {len(values)}.")
    
    if not all(math.isfinite(v) for v in values):
        raise ValueError("EMA values must all be finite numbers.")
    
    result = sum(values[:period]) / period
    multiplier = 2 / (period + 1)
    for value in values[period:]:
        if not math.isfinite(value):
            raise ValueError(f"EMA encountered non-finite value: {value}")
        result = value * multiplier + result * (1 - multiplier)
    
    if not math.isfinite(result):
        raise ValueError(f"EMA calculation resulted in non-finite value: {result}")
    return result


def calculate_indicators(candles):
    closes = [candle.close for candle in candles]
    if len(closes) < MIN_CANDLES_BUFFER:
        raise ValueError(f"Not enough candles to calculate the strategy indicators: need {MIN_CANDLES_BUFFER}, got {len(closes)}.")

    gains = []
    losses = []
    for previous, current in zip(closes[-RSI_PERIOD - 1 : -1], closes[-RSI_PERIOD:]):
        change = current - previous
        gains.append(max(change, 0))
        losses.append(max(-change, 0))
    
    average_gain = sum(gains) / RSI_PERIOD
    average_loss = sum(losses) / RSI_PERIOD
    
    if average_loss == 0:
        rsi = 100.0 if average_gain > 0 else 50.0
    else:
        rs = average_gain / average_loss
        if not math.isfinite(rs) or rs < 0:
            raise ValueError(f"RSI calculation resulted in invalid RS value: {rs}")
        rsi = 100 - 100 / (1 + rs)
    
    if not 0 <= rsi <= 100:
        raise ValueError(f"RSI value out of valid range [0, 100]: {rsi}")

    true_ranges = []
    for previous, candle in zip(candles[-ATR_PERIOD - 1 : -1], candles[-ATR_PERIOD:]):
        tr = max(
            candle.high - candle.low,
            abs(candle.high - previous.close),
            abs(candle.low - previous.close),
        )
        if not math.isfinite(tr) or tr < 0:
            raise ValueError(f"True range calculation resulted in invalid value: {tr}")
        true_ranges.append(tr)
    
    atr = sum(true_ranges) / ATR_PERIOD
    if not math.isfinite(atr):
        raise ValueError(f"ATR calculation resulted in non-finite value: {atr}")
    
    return Indicators(
        fast_ema=ema(closes, FAST_EMA),
        slow_ema=ema(closes, SLOW_EMA),
        rsi=rsi,
        atr=atr,
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
    except json.JSONDecodeError as error:
        raise RuntimeError(f"State file is not valid JSON: {path}. Error: {error}") from error
    except (IOError, OSError) as error:
        raise RuntimeError(f"Failed to read state file: {path}. Error: {error}") from error
    
    if not isinstance(state, dict):
        raise RuntimeError(f"Invalid state file format (expected dict): {path}")
    
    if state.get("version") != 1:
        raise RuntimeError(f"Unsupported state version in {path}: {state.get('version')}. Expected version 1. Preserve or rename it before restarting.")
    
    state = {**default_state(), **state}
    
    if state["symbol"] != SYMBOL or state["interval"] != INTERVAL:
        raise RuntimeError(
            f"State file {path} belongs to {state['symbol']} / {state['interval']}, "
            f"not {SYMBOL} / {INTERVAL}. The state must match the current configuration."
        )
    
    if not isinstance(state["in_position"], bool):
        raise RuntimeError(f"Invalid position flag in state file: {path}. Expected boolean, got {type(state['in_position'])}")
    
    if not isinstance(state["last_candle_time"], int) or state["last_candle_time"] < 0:
        raise RuntimeError(f"Invalid last_candle_time in state file: {path}. Expected non-negative integer, got {state['last_candle_time']}")
    
    for key in (
        "quantity",
        "entry_price",
        "quote_spent",
        "stop_price",
        "take_profit_price",
    ):
        value = state[key]
        if not isinstance(value, (int, float)):
            raise RuntimeError(f"Invalid {key} in state file: {path}. Expected number, got {type(value)}")
        if not math.isfinite(value):
            raise RuntimeError(f"Invalid {key} in state file: {path}. Expected finite number, got {value}")
        if value < 0:
            raise RuntimeError(f"Invalid {key} in state file: {path}. Expected non-negative, got {value}")
    
    if state["in_position"]:
        if not (state["quantity"] > 0 and state["entry_price"] > 0 and state["quote_spent"] > 0):
            raise RuntimeError(f"Inconsistent open position in state file: {path}. Position requires positive quantity, entry_price, and quote_spent")
        if not (state["stop_price"] > 0 and state["take_profit_price"] > state["stop_price"]):
            raise RuntimeError(f"Inconsistent stop/target levels in state file: {path}. Requires 0 < stop_price < take_profit_price")
    
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

    if not math.isfinite(candle.close):
        raise ValueError(f"Candle close price is not finite: {candle.close}")

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
            if price <= 0 or not math.isfinite(price):
                raise ValueError(f"Invalid exit price calculated: {price}")
            
            proceeds = state["quantity"] * price * (1 - FEE_RATE)
            pnl = proceeds - state["quote_spent"]
            
            if not math.isfinite(proceeds) or not math.isfinite(pnl):
                raise ValueError(f"Invalid PnL calculation: proceeds={proceeds}, pnl={pnl}")
            
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
        if price <= 0 or not math.isfinite(price):
            raise ValueError(f"Invalid entry price: {price}")
        
        quantity = QUOTE_ORDER_SIZE * (1 - FEE_RATE) / price
        
        if quantity <= 0 or not math.isfinite(quantity):
            raise ValueError(f"Invalid quantity calculated: {quantity}")
        
        stop_price = max(price - 2 * indicators.atr, price * 0.001)
        take_profit_price = price + 3 * indicators.atr
        
        if not (0 < stop_price < take_profit_price):
            raise ValueError(f"Invalid stop/target levels: stop={stop_price}, target={take_profit_price}")
        
        state = {
            **state,
            "in_position": True,
            "quantity": quantity,
            "entry_price": price,
            "quote_spent": QUOTE_ORDER_SIZE,
            "stop_price": stop_price,
            "take_profit_price": take_profit_price,
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
        raise RuntimeError("Safety stop: ENABLE_LIVE_TRADING must remain NO. This prevents accidental live trading.")
    if not SYMBOL.endswith("USDT"):
        raise RuntimeError(f"Use a USDT-quoted symbol; got {SYMBOL}.")
    if not math.isfinite(QUOTE_ORDER_SIZE) or QUOTE_ORDER_SIZE <= 0:
        raise RuntimeError(f"QUOTE_ORDER_SIZE must be positive finite number; got {QUOTE_ORDER_SIZE}.")
    if INTERVAL not in INTERVAL_SECONDS:
        raise RuntimeError(f"Unsupported candle interval: {INTERVAL}. Supported intervals: {list(INTERVAL_SECONDS.keys())}")
    if not math.isfinite(FEE_RATE) or not (0 <= FEE_RATE < 0.05):
        raise RuntimeError(f"PAPER_FEE_RATE must be between 0 and 0.05; got {FEE_RATE}.")
    if not math.isfinite(NETWORK_TIMEOUT) or NETWORK_TIMEOUT <= 0:
        raise RuntimeError(f"NETWORK_TIMEOUT must be positive; got {NETWORK_TIMEOUT}.")
    if not isinstance(MAX_RETRIES, int) or MAX_RETRIES < 1:
        raise RuntimeError(f"MAX_RETRIES must be at least 1; got {MAX_RETRIES}.")

    try:
        candles = fetch_closed_candles()
        indicators = calculate_indicators(candles)
    except (RuntimeError, ValueError) as error:
        raise RuntimeError(f"Failed to fetch or process market data: {error}") from error
    
    try:
        state = load_state()
    except RuntimeError as error:
        raise RuntimeError(f"Failed to load trading state: {error}") from error
    
    candle = candles[-1]
    if candle.close_time <= state["last_candle_time"]:
        print("Closed candle already recorded; no duplicate evaluation.")
        return
    
    try:
        new_state, trade = process_candle(state, candles[-1], indicators)
    except (ValueError, RuntimeError) as error:
        raise RuntimeError(f"Failed to process candle: {error}") from error
    
    try:
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
    except (RuntimeError, OSError) as error:
        raise RuntimeError(f"Failed to append event to ledger: {error}") from error
    
    try:
        save_state(new_state)
    except (RuntimeError, OSError) as error:
        raise RuntimeError(f"Failed to save trading state: {error}") from error
    
    print(
        f"{SYMBOL} {INTERVAL} close={candle.close:.12g} "
        f"EMA{FAST_EMA}={indicators.fast_ema:.12g} "
        f"EMA{SLOW_EMA}={indicators.slow_ema:.12g} "
        f"RSI{RSI_PERIOD}={indicators.rsi:.2f} ATR{ATR_PERIOD}={indicators.atr:.12g}"
    )
    if trade:
        try:
            append_trade(**trade)
        except (RuntimeError, OSError) as error:
            raise RuntimeError(f"Failed to append trade to CSV: {error}") from error
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
    
    try:
        if args.verify_ledger:
            result = verify_ledger(LEDGER_FILE)
            print(json.dumps(result, indent=2))
            return 0
        if args.import_binance_csv:
            if not Path(args.import_binance_csv).exists():
                raise ValueError(f"CSV file not found: {args.import_binance_csv}")
            result = import_binance_csv(args.import_binance_csv, LEDGER_FILE)
            print(json.dumps(result, indent=2))
            return 0
        if args.backtest or args.walk_forward:
            from binance_backtest import backtest, load_candles_csv, walk_forward

            csv_path = args.backtest or args.walk_forward
            if not Path(csv_path).exists():
                raise ValueError(f"CSV file not found: {csv_path}")
            
            candles = load_candles_csv(csv_path)
            if not candles:
                raise ValueError(f"No candles loaded from {csv_path}")
            
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
            
            if args.initial_cash <= 0:
                raise ValueError(f"Initial cash must be positive; got {args.initial_cash}")
            if args.spread_bps < 0 or args.slippage_bps < 0:
                raise ValueError(f"Spread and slippage must be non-negative")
            if args.folds < 2:
                raise ValueError(f"Number of folds must be at least 2; got {args.folds}")
            
            result = (
                backtest(candles, SLOW_EMA, len(candles), **options)
                if args.backtest
                else walk_forward(candles, folds=args.folds, **options)
            )
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
    except (RuntimeError, ValueError, OSError) as error:
        print(f"Fatal error: {error}", file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print("\nShutdown requested by user.", file=sys.stderr)
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
