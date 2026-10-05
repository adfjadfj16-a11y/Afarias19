"""Chronological, paper-only backtesting for the fixed baseline strategy."""

import csv
import math
import re
from pathlib import Path

from bot_spot_binance_safe import (
    ATR_PERIOD,
    FAST_EMA,
    INTERVAL_SECONDS,
    RSI_PERIOD,
    SLOW_EMA,
    Candle,
    Indicators,
)


def _header_map(fieldnames):
    return {
        re.sub(r"[^a-z0-9]", "", (name or "").lower()): name
        for name in (fieldnames or [])
    }


def _column(headers, *names):
    for name in names:
        key = re.sub(r"[^a-z0-9]", "", name.lower())
        if key in headers:
            return headers[key]
    raise ValueError(f"OHLCV CSV is missing required column: {names[0]}.")


def load_candles_csv(path):
    candles = []
    with Path(path).open(newline="", encoding="utf-8-sig") as source:
        reader = csv.DictReader(source)
        headers = _header_map(reader.fieldnames)
        time_col = _column(headers, "timestamp", "open_time", "date(utc)", "date", "time")
        columns = {
            "open": _column(headers, "open"),
            "high": _column(headers, "high"),
            "low": _column(headers, "low"),
            "close": _column(headers, "close"),
        }
        for line, row in enumerate(reader, 2):
            try:
                timestamp = row[time_col].strip()
                if timestamp.isdigit():
                    timestamp_ms = int(timestamp)
                    if timestamp_ms < 10_000_000_000:
                        timestamp_ms *= 1000
                else:
                    from datetime import datetime

                    parsed = datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
                    if parsed.tzinfo is None:
                        raise ValueError("Timestamp must include a timezone.")
                    timestamp_ms = int(parsed.timestamp() * 1000)
                values = {
                    name: float(row[column])
                    for name, column in columns.items()
                }
                candle = Candle(
                    open_time=timestamp_ms,
                    open=values["open"],
                    high=values["high"],
                    low=values["low"],
                    close=values["close"],
                    close_time=timestamp_ms,
                )
            except (AttributeError, KeyError, TypeError, ValueError) as error:
                raise ValueError(f"Invalid OHLCV data on CSV line {line}.") from error
            if not all(math.isfinite(price) and price > 0 for price in values.values()):
                raise ValueError(f"Invalid OHLCV price on CSV line {line}.")
            if (
                candle.high < candle.low
                or not candle.low <= candle.open <= candle.high
                or not candle.low <= candle.close <= candle.high
            ):
                raise ValueError(f"Inconsistent OHLC values on CSV line {line}.")
            if candles and candle.open_time <= candles[-1].open_time:
                raise ValueError("OHLCV data must be strictly chronological with no duplicates.")
            candles.append(candle)
    if not candles:
        raise ValueError("OHLCV CSV contains no candles.")
    return candles


def _indicator_series(candles):
    closes = []
    fast_ema = slow_ema = None
    alpha_fast = 2 / (FAST_EMA + 1)
    alpha_slow = 2 / (SLOW_EMA + 1)
    indicators = []
    for index, candle in enumerate(candles):
        closes.append(candle.close)
        if index == FAST_EMA - 1:
            fast_ema = sum(closes[:FAST_EMA]) / FAST_EMA
        elif index >= FAST_EMA:
            fast_ema = candle.close * alpha_fast + fast_ema * (1 - alpha_fast)
        if index == SLOW_EMA - 1:
            slow_ema = sum(closes[:SLOW_EMA]) / SLOW_EMA
        elif index >= SLOW_EMA:
            slow_ema = candle.close * alpha_slow + slow_ema * (1 - alpha_slow)
        if index < SLOW_EMA:
            indicators.append(None)
            continue

        changes = [
            closes[position] - closes[position - 1]
            for position in range(index - RSI_PERIOD + 1, index + 1)
        ]
        gains = sum(max(change, 0) for change in changes) / RSI_PERIOD
        losses = sum(max(-change, 0) for change in changes) / RSI_PERIOD
        rsi = 100.0 if losses == 0 and gains else (
            50.0 if losses == 0 else 100 - 100 / (1 + gains / losses)
        )
        true_ranges = []
        for position in range(index - ATR_PERIOD + 1, index + 1):
            current = candles[position]
            previous_close = candles[position - 1].close
            true_ranges.append(
                max(
                    current.high - current.low,
                    abs(current.high - previous_close),
                    abs(current.low - previous_close),
                )
            )
        indicators.append(
            Indicators(
                fast_ema=fast_ema,
                slow_ema=slow_ema,
                rsi=rsi,
                atr=sum(true_ranges) / ATR_PERIOD,
            )
        )
    return indicators


def _cost_rate(commission, spread_bps, slippage_bps):
    if not all(math.isfinite(value) for value in (commission, spread_bps, slippage_bps)):
        raise ValueError("Cost assumptions must be finite numbers.")
    if commission < 0 or commission >= 0.05 or spread_bps < 0 or slippage_bps < 0:
        raise ValueError("Costs must be non-negative; commission must be below 5%.")
    return commission + spread_bps / 20_000 + slippage_bps / 10_000


def _metrics(equities, initial_cash, benchmark_return, interval):
    if len(equities) < 2:
        raise ValueError("Test period must contain at least two candles.")
    returns = []
    previous = initial_cash
    for value in equities:
        returns.append(value / previous - 1 if previous else 0.0)
        previous = value
    peak = initial_cash
    max_drawdown = 0.0
    for value in equities:
        peak = max(peak, value)
        if peak:
            max_drawdown = max(max_drawdown, (peak - value) / peak)
    bars_per_year = 365.25 * 24 * 3600 / INTERVAL_SECONDS.get(interval, 86400)
    mean_return = sum(returns) / len(returns)
    variance = sum((value - mean_return) ** 2 for value in returns) / len(returns)
    volatility = math.sqrt(variance * bars_per_year)
    final_equity = equities[-1]
    return {
        "bars": len(equities),
        "final_equity": final_equity,
        "total_return_pct": (final_equity / initial_cash - 1) * 100,
        "buy_and_hold_return_pct": benchmark_return * 100,
        "excess_vs_buy_and_hold_pct": (final_equity / initial_cash - 1 - benchmark_return) * 100,
        "max_drawdown_pct": max_drawdown * 100,
        "annualized_volatility_pct": volatility * 100,
    }


def backtest(
    candles,
    start_index,
    end_index,
    *,
    initial_cash=1000.0,
    quote_size=5.0,
    commission=0.001,
    spread_bps=2.0,
    slippage_bps=2.0,
    interval="5m",
):
    if not isinstance(start_index, int) or not isinstance(end_index, int):
        raise ValueError("Backtest indices must be integers.")
    if start_index < SLOW_EMA or end_index > len(candles) or end_index - start_index < 2:
        raise ValueError("Backtest needs a valid period with indicator warm-up and at least two bars.")
    if not math.isfinite(initial_cash) or initial_cash <= 0:
        raise ValueError("Initial cash must be a positive finite amount.")
    if not math.isfinite(quote_size) or quote_size <= 0:
        raise ValueError("Position size must be a positive finite amount.")
    cost_rate = _cost_rate(commission, spread_bps, slippage_bps)
    indicator_series = _indicator_series(candles[:end_index])
    cash = initial_cash
    quantity = 0.0
    stop_price = take_profit_price = 0.0
    pending_action = None
    equities = []
    trades = 0
    test_candles = candles[start_index:end_index]

    for index in range(start_index, end_index):
        candle = candles[index]
        prior_indicators = indicator_series[index - 1] if index else None

        if pending_action == "BUY" and quantity == 0:
            budget = min(quote_size, cash)
            if budget > 0 and prior_indicators and prior_indicators.atr > 0:
                fill_price = candle.open * (1 + cost_rate)
                quantity = budget / fill_price
                cash -= budget
                stop_price = max(fill_price - 2 * prior_indicators.atr, fill_price * 0.001)
                take_profit_price = fill_price + 3 * prior_indicators.atr
                trades += 1
        elif pending_action == "SELL" and quantity > 0:
            cash += quantity * candle.open * (1 - cost_rate)
            quantity = 0.0
            trades += 1
        pending_action = None

        if quantity > 0:
            stop_hit = candle.low <= stop_price
            target_hit = candle.high >= take_profit_price
            if stop_hit or target_hit:
                if stop_hit:
                    raw_fill = min(stop_price, candle.open)
                else:
                    raw_fill = max(take_profit_price, candle.open)
                cash += quantity * raw_fill * (1 - cost_rate)
                quantity = 0.0
                trades += 1

        mark_value = cash + quantity * candle.close * (1 - cost_rate)
        equities.append(mark_value)

        current_indicators = indicator_series[index]
        if current_indicators:
            if quantity > 0 and (
                current_indicators.fast_ema < current_indicators.slow_ema
                or current_indicators.rsi < 45
            ):
                pending_action = "SELL"
            elif quantity == 0 and (
                current_indicators.fast_ema > current_indicators.slow_ema
                and 50 <= current_indicators.rsi <= 68
                and current_indicators.atr > 0
            ):
                pending_action = "BUY"

    final_candle = candles[end_index - 1]
    final_equity = cash + quantity * final_candle.close * (1 - cost_rate)
    equities[-1] = final_equity

    first_price = test_candles[0].open * (1 + cost_rate)
    benchmark_quantity = initial_cash / first_price
    benchmark_final = benchmark_quantity * final_candle.close * (1 - cost_rate)
    benchmark_return = benchmark_final / initial_cash - 1
    result = _metrics(equities, initial_cash, benchmark_return, interval)
    result["completed_trades"] = trades
    result["open_position_at_end"] = quantity > 0
    return result


def walk_forward(
    candles,
    *,
    train_fraction=0.6,
    holdout_fraction=0.2,
    folds=4,
    **backtest_options,
):
    if not 0 < train_fraction < 1 or not 0 < holdout_fraction < 1:
        raise ValueError("Train and holdout fractions must each be between 0 and 1.")
    if train_fraction + holdout_fraction >= 1:
        raise ValueError("Train plus holdout must leave room for walk-forward folds.")
    if not isinstance(folds, int) or folds < 1:
        raise ValueError("Walk-forward folds must be a positive integer.")
    train_end = int(len(candles) * train_fraction)
    holdout_start = int(len(candles) * (1 - holdout_fraction))
    if train_end <= SLOW_EMA or holdout_start - train_end < folds or len(candles) - holdout_start < 2:
        raise ValueError("Insufficient candles for warm-up, walk-forward folds, and final holdout.")

    train = backtest(candles, SLOW_EMA, train_end, **backtest_options)
    fold_results = []
    validation_size = holdout_start - train_end
    for fold in range(folds):
        start = train_end + validation_size * fold // folds
        end = train_end + validation_size * (fold + 1) // folds
        if end - start < 2:
            raise ValueError("Each walk-forward fold needs at least two candles.")
        fold_results.append(
            {
                "fold": fold + 1,
                "start_index": start,
                "end_index": end,
                "metrics": backtest(candles, start, end, **backtest_options),
            }
        )
    holdout = backtest(candles, holdout_start, len(candles), **backtest_options)
    return {
        "method": "chronological fixed-strategy evaluation; no parameter optimization",
        "train": {"start_index": SLOW_EMA, "end_index": train_end, "metrics": train},
        "walk_forward": fold_results,
        "final_holdout": {
            "start_index": holdout_start,
            "end_index": len(candles),
            "metrics": holdout,
            "used_for_tuning": False,
        },
    }
