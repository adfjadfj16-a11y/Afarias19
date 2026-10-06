#!/usr/bin/env python3
"""Perpetual futures scalping bot. Detects micro-trends and volatility for quick trades.

Educational tool for USDT-M futures on Binance Testnet. Performs no actual orders.
Tracks scalp opportunities with complete persistence and pattern analysis.
"""

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
from collections import defaultdict
from dataclasses import dataclass, asdict
from datetime import datetime, timezone, timedelta
from pathlib import Path
from statistics import mean, stdev

from private_ledger import (
    append_event,
    default_data_dir,
    ensure_outside_repository,
    secure_mkdir,
    verify_ledger,
)

TESTNET_BASE_URL = "https://testnet.binance.vision"
SYMBOL = os.getenv("SYMBOL", "BNBUSDT").upper()
INTERVAL = os.getenv("INTERVAL", "1m")
INTERVAL_SECONDS = {
    "1m": 60,
    "3m": 180,
    "5m": 300,
    "15m": 900,
    "30m": 1800,
    "1h": 3600,
}
NETWORK_TIMEOUT = float(os.getenv("NETWORK_TIMEOUT", "5"))
MAX_RETRIES = int(os.getenv("MAX_RETRIES", "3"))
RETRY_DELAY = float(os.getenv("RETRY_DELAY", "1"))

# Scalping parameters
POSITION_SIZE_USDT = float(os.getenv("POSITION_SIZE_USDT", "50"))
SCALP_TARGET_BPS = float(os.getenv("SCALP_TARGET_BPS", "10"))
SCALP_STOP_BPS = float(os.getenv("SCALP_STOP_BPS", "5"))
VOLATILITY_THRESHOLD = float(os.getenv("VOLATILITY_THRESHOLD", "0.8"))
RSI_OVERSOLD = float(os.getenv("RSI_OVERSOLD", "30"))
RSI_OVERBOUGHT = float(os.getenv("RSI_OVERBOUGHT", "70"))
MAX_CONCURRENT_SCALPS = int(os.getenv("MAX_CONCURRENT_SCALPS", "3"))
LEVERAGE = float(os.getenv("LEVERAGE", "5"))

# Data management
DATA_DIR = default_data_dir()
SCALP_FILE = Path(os.getenv("SCALP_FILE", DATA_DIR / "scalps_perpetual.jsonl"))
SCALP_PATTERN_FILE = Path(os.getenv("SCALP_PATTERN_FILE", DATA_DIR / "scalp_patterns.json"))
LEDGER_FILE = Path(os.getenv("LEDGER_FILE", DATA_DIR / "investment-ledger.jsonl"))


@dataclass(frozen=True)
class Candle:
    open_time: int
    open: float
    high: float
    low: float
    close: float
    volume: float
    close_time: int


@dataclass(frozen=True)
class ScalpIndicators:
    rsi: float
    atr: float
    volatility: float
    trend_ema: float
    current_price: float


@dataclass(frozen=True)
class ScalpOpportunity:
    timestamp: int
    symbol: str
    direction: str  # "LONG" or "SHORT"
    entry_price: float
    target_price: float
    stop_price: float
    position_size: float
    reason: str


@dataclass(frozen=True)
class CompletedScalp:
    timestamp: int
    symbol: str
    direction: str
    entry_time: int
    entry_price: float
    exit_time: int
    exit_price: float
    position_size: float
    pnl_usdt: float
    pnl_percent: float
    duration_seconds: int
    exit_reason: str
    target_hit: bool


def fetch_klines(symbol=SYMBOL, interval=INTERVAL, limit=100):
    """Fetch K-line candles from Binance Testnet with retry logic."""
    query = urllib.parse.urlencode(
        {"symbol": symbol, "interval": interval, "limit": limit}
    )
    url = f"{TESTNET_BASE_URL}/api/v3/klines?{query}"
    request = urllib.request.Request(
        url,
        headers={"User-Agent": "perpetual-scalper/1.0"},
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
            raise RuntimeError(f"Could not fetch klines after {MAX_RETRIES} attempts: {error}") from error

    if not isinstance(rows, list):
        raise RuntimeError("Unexpected klines response from Binance Testnet: not a list.")
    
    candles = []
    try:
        for row in rows:
            if not isinstance(row, (list, tuple)) or len(row) < 9:
                raise ValueError(f"Invalid candle row format: {row}")
            
            candle = Candle(
                open_time=int(row[0]),
                open=float(row[1]),
                high=float(row[2]),
                low=float(row[3]),
                close=float(row[4]),
                volume=float(row[7]),
                close_time=int(row[6]),
            )
            
            # Validate candle data
            if not all(
                math.isfinite(v) and v > 0
                for v in (candle.open, candle.high, candle.low, candle.close, candle.volume)
            ):
                raise ValueError(f"Candle contains invalid prices: {candle}")
            
            if not (candle.low <= candle.open <= candle.high and 
                    candle.low <= candle.close <= candle.high):
                raise ValueError(f"Candle OHLC inconsistent: {candle}")
            
            candles.append(candle)
    
    except (IndexError, TypeError, ValueError) as error:
        raise RuntimeError(f"Invalid candle data: {error}") from error
    
    if len(candles) < 50:
        raise RuntimeError(f"Need at least 50 candles; received {len(candles)}.")
    
    if any(a.open_time >= b.open_time for a, b in zip(candles, candles[1:])):
        raise RuntimeError("Candle timestamps are not strictly increasing.")
    
    return candles


def calculate_rsi(prices, period=14):
    """Calculate RSI indicator."""
    if len(prices) < period + 1:
        raise ValueError(f"Not enough prices for RSI: need {period + 1}, got {len(prices)}")
    
    gains, losses = [], []
    for i in range(1, period + 1):
        change = prices[-period - 1 + i] - prices[-period - 2 + i]
        gains.append(max(change, 0))
        losses.append(max(-change, 0))
    
    avg_gain = sum(gains) / period
    avg_loss = sum(losses) / period
    
    if avg_loss == 0:
        return 100.0 if avg_gain > 0 else 50.0
    
    rs = avg_gain / avg_loss
    rsi = 100 - 100 / (1 + rs)
    return rsi


def calculate_atr(candles, period=14):
    """Calculate Average True Range."""
    if len(candles) < period + 1:
        raise ValueError(f"Not enough candles for ATR: need {period + 1}, got {len(candles)}")
    
    true_ranges = []
    for i in range(len(candles) - period, len(candles)):
        prev = candles[i - 1]
        curr = candles[i]
        tr = max(
            curr.high - curr.low,
            abs(curr.high - prev.close),
            abs(curr.low - prev.close),
        )
        true_ranges.append(tr)
    
    atr = sum(true_ranges) / period
    if not math.isfinite(atr) or atr < 0:
        raise ValueError(f"Invalid ATR calculated: {atr}")
    return atr


def calculate_volatility(candles, period=20):
    """Calculate volatility as standard deviation of returns."""
    if len(candles) < period + 1:
        raise ValueError(f"Not enough candles for volatility: need {period + 1}, got {len(candles)}")
    
    returns = []
    for i in range(-period, 0):
        ret = (candles[i].close - candles[i - 1].close) / candles[i - 1].close
        returns.append(ret)
    
    if len(returns) < 2:
        return 0.0
    
    vol = stdev(returns) if len(returns) > 1 else 0.0
    return vol


def calculate_ema(prices, period):
    """Calculate Exponential Moving Average."""
    if len(prices) < period:
        raise ValueError(f"Not enough values for EMA: need {period}, got {len(prices)}")
    
    ema = sum(prices[-period:]) / period
    multiplier = 2 / (period + 1)
    
    for price in prices[-period:-1]:
        ema = price * multiplier + ema * (1 - multiplier)
    
    if not math.isfinite(ema):
        raise ValueError(f"Invalid EMA: {ema}")
    return ema


def analyze_scalp_indicators(candles):
    """Calculate all indicators for scalp analysis."""
    if len(candles) < 50:
        raise ValueError("Not enough candles for indicator analysis")
    
    closes = [c.close for c in candles]
    
    rsi = calculate_rsi(closes, 14)
    atr = calculate_atr(candles, 14)
    volatility = calculate_volatility(candles, 20)
    trend_ema = calculate_ema(closes, 50)
    current_price = closes[-1]
    
    if not all(math.isfinite(v) for v in (rsi, atr, volatility, trend_ema)):
        raise ValueError("Non-finite indicator values calculated")
    
    return ScalpIndicators(
        rsi=rsi,
        atr=atr,
        volatility=volatility,
        trend_ema=trend_ema,
        current_price=current_price,
    )


def detect_scalp_opportunity(candles, indicators, active_scalps):
    """Detect scalping opportunities based on indicators and risk management."""
    if len(active_scalps) >= MAX_CONCURRENT_SCALPS:
        return None
    
    price = indicators.current_price
    atr = indicators.atr
    volatility = indicators.volatility
    
    if volatility < VOLATILITY_THRESHOLD or atr <= 0:
        return None
    
    target_distance = price * (SCALP_TARGET_BPS / 10000)
    stop_distance = price * (SCALP_STOP_BPS / 10000)
    position_size = (POSITION_SIZE_USDT * LEVERAGE) / price
    
    timestamp = int(time.time() * 1000)
    
    # Long scalp: RSI oversold, price above EMA trend
    if indicators.rsi < RSI_OVERSOLD and price > indicators.trend_ema:
        if volatility > VOLATILITY_THRESHOLD:
            return ScalpOpportunity(
                timestamp=timestamp,
                symbol=SYMBOL,
                direction="LONG",
                entry_price=price,
                target_price=price + target_distance,
                stop_price=price - stop_distance,
                position_size=position_size,
                reason=f"RSI oversold ({indicators.rsi:.2f}), price above EMA"
            )
    
    # Short scalp: RSI overbought, price below EMA trend
    elif indicators.rsi > RSI_OVERBOUGHT and price < indicators.trend_ema:
        if volatility > VOLATILITY_THRESHOLD:
            return ScalpOpportunity(
                timestamp=timestamp,
                symbol=SYMBOL,
                direction="SHORT",
                entry_price=price,
                target_price=price - target_distance,
                stop_price=price + stop_distance,
                position_size=position_size,
                reason=f"RSI overbought ({indicators.rsi:.2f}), price below EMA"
            )
    
    return None


def process_scalp_exit(scalp, current_price, candle_time):
    """Determine if scalp should exit and generate completion record."""
    if scalp['direction'] == "LONG":
        target_hit = current_price >= scalp['target_price']
        stop_hit = current_price <= scalp['stop_price']
        exit_price = current_price
    else:  # SHORT
        target_hit = current_price <= scalp['target_price']
        stop_hit = current_price >= scalp['stop_price']
        exit_price = current_price
    
    if not (target_hit or stop_hit):
        return None  # No exit yet
    
    # Choose better exit price
    if target_hit and stop_hit:
        exit_price = scalp['target_price']  # Target has priority
    elif target_hit:
        exit_price = scalp['target_price']
    else:
        exit_price = scalp['stop_price']
    
    duration = (candle_time - scalp['entry_time']) // 1000
    pnl_per_unit = (exit_price - scalp['entry_price']) if scalp['direction'] == "LONG" else (scalp['entry_price'] - exit_price)
    pnl_usdt = pnl_per_unit * scalp['position_size']
    pnl_percent = (pnl_per_unit / scalp['entry_price']) * 100
    
    return CompletedScalp(
        timestamp=int(time.time() * 1000),
        symbol=SYMBOL,
        direction=scalp['direction'],
        entry_time=scalp['entry_time'],
        entry_price=scalp['entry_price'],
        exit_time=candle_time,
        exit_price=exit_price,
        position_size=scalp['position_size'],
        pnl_usdt=pnl_usdt,
        pnl_percent=pnl_percent,
        duration_seconds=duration,
        exit_reason="TARGET" if target_hit else "STOP",
        target_hit=target_hit,
    )


def save_completed_scalp(scalp, path=SCALP_FILE):
    """Persist completed scalp to CSV."""
    path = ensure_outside_repository(path)
    secure_mkdir(path.parent)
    is_new = not path.exists()
    
    try:
        with path.open("a", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=[
                "timestamp", "symbol", "direction", "entry_time", "entry_price",
                "exit_time", "exit_price", "position_size", "pnl_usdt", "pnl_percent",
                "duration_seconds", "exit_reason", "target_hit"
            ])
            if is_new:
                writer.writeheader()
            writer.writerow(asdict(scalp))
    except (OSError, IOError) as error:
        raise RuntimeError(f"Failed to save scalp: {error}") from error


def analyze_scalp_patterns(path=SCALP_FILE):
    """Analyze patterns in completed scalps."""
    if not path.exists():
        return {
            "total_scalps": 0,
            "win_rate": 0.0,
            "avg_pnl": 0.0,
            "total_pnl": 0.0,
            "avg_duration_seconds": 0,
        }
    
    scalps = []
    try:
        with path.open("r", encoding="utf-8") as f:
            reader = csv.DictReader(f)
            for row in reader:
                scalps.append({
                    "pnl_usdt": float(row["pnl_usdt"]),
                    "pnl_percent": float(row["pnl_percent"]),
                    "duration_seconds": int(row["duration_seconds"]),
                    "direction": row["direction"],
                    "target_hit": row["target_hit"] == "True",
                })
    except (OSError, IOError, ValueError) as error:
        raise RuntimeError(f"Failed to analyze patterns: {error}") from error
    
    if not scalps:
        return {
            "total_scalps": 0,
            "win_rate": 0.0,
            "avg_pnl": 0.0,
            "total_pnl": 0.0,
            "avg_duration_seconds": 0,
        }
    
    winners = sum(1 for s in scalps if s["pnl_usdt"] > 0)
    total_pnl = sum(s["pnl_usdt"] for s in scalps)
    avg_pnl = total_pnl / len(scalps)
    avg_duration = sum(s["duration_seconds"] for s in scalps) // len(scalps) if scalps else 0
    
    return {
        "total_scalps": len(scalps),
        "win_rate": (winners / len(scalps) * 100) if scalps else 0.0,
        "avg_pnl": avg_pnl,
        "total_pnl": total_pnl,
        "avg_duration_seconds": avg_duration,
        "winners": winners,
        "losers": len(scalps) - winners,
    }


def run_once():
    """Execute one scalping analysis cycle."""
    if os.getenv("BINANCE_TESTNET") != "1":
        raise RuntimeError("Set BINANCE_TESTNET=1. Only Binance Testnet is supported.")
    if os.getenv("ENABLE_LIVE_TRADING", "NO").upper() != "NO":
        raise RuntimeError("Safety stop: ENABLE_LIVE_TRADING must remain NO.")
    
    if not SYMBOL.endswith("USDT"):
        raise RuntimeError(f"Use a USDT-quoted symbol; got {SYMBOL}.")
    if INTERVAL not in INTERVAL_SECONDS:
        raise RuntimeError(f"Unsupported interval: {INTERVAL}")
    
    try:
        candles = fetch_klines()
        indicators = analyze_scalp_indicators(candles)
    except (RuntimeError, ValueError) as error:
        raise RuntimeError(f"Failed to fetch/analyze market data: {error}") from error
    
    try:
        opportunity = detect_scalp_opportunity(candles, indicators, [])
    except (RuntimeError, ValueError) as error:
        raise RuntimeError(f"Failed to detect opportunity: {error}") from error
    
    current_candle = candles[-1]
    print(
        f"{SYMBOL} {INTERVAL}: price={indicators.current_price:.8g} "
        f"RSI={indicators.rsi:.2f} ATR={indicators.atr:.8g} VOL={indicators.volatility:.4f}"
    )
    
    if opportunity:
        print(f"SCALP SIGNAL ({opportunity.direction}): {opportunity.reason}")
        try:
            append_event(
                LEDGER_FILE,
                "scalp_signal",
                {
                    "event_id": f"scalp:{SYMBOL}:{current_candle.close_time}:{opportunity.direction}",
                    "timestamp": opportunity.timestamp,
                    "symbol": SYMBOL,
                    "direction": opportunity.direction,
                    "entry_price": opportunity.entry_price,
                    "target_price": opportunity.target_price,
                    "stop_price": opportunity.stop_price,
                    "position_size": opportunity.position_size,
                    "reason": opportunity.reason,
                }
            )
        except (RuntimeError, OSError) as error:
            print(f"Warning: Failed to log scalp signal: {error}", file=sys.stderr)
    else:
        print("No scalp opportunity detected.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--loop", action="store_true", help="Run continuously.")
    parser.add_argument("--analyze", action="store_true", help="Analyze scalp patterns and exit.")
    parser.add_argument("--verify-ledger", action="store_true", help="Verify ledger integrity.")
    args = parser.parse_args()
    
    try:
        if args.verify_ledger:
            result = verify_ledger(LEDGER_FILE)
            print(json.dumps(result, indent=2))
            return 0
        
        if args.analyze:
            analysis = analyze_scalp_patterns(SCALP_FILE)
            print(json.dumps(analysis, indent=2))
            return 0
        
        while True:
            try:
                run_once()
            except (RuntimeError, OSError, ValueError) as error:
                print(f"Error: {error}", file=sys.stderr)
                return 1
            
            if not args.loop:
                return 0
            
            time.sleep(INTERVAL_SECONDS[INTERVAL])
    
    except KeyboardInterrupt:
        print("\nShutdown requested.", file=sys.stderr)
        return 0
    except (RuntimeError, ValueError) as error:
        print(f"Fatal error: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
