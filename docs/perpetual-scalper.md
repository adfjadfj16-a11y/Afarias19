# Perpetual Futures Scalping Bot

`bot_perpetual_scalper.py` is an educational scalping bot for USDT-M perpetual futures on Binance Testnet. It detects micro-trend opportunities and executes paper trades with complete persistence and pattern analysis. **It never submits actual orders.**

## Features

- **Scalp Detection**: Identifies oversold/overbought opportunities using RSI, ATR, and volatility
- **Pattern Analysis**: Tracks all scalps with entry/exit prices, PnL, duration, and exit reason
- **Risk Management**: Configurable position size, stop-loss, target levels, and maximum concurrent positions
- **Persistent Ledger**: Every scalp is recorded for later analysis and pattern recognition
- **Safety Guards**: Requires explicit testnet flag and blocks accidental live trading

## Run

From repository root, use Python 3.10+:

```bash
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py
```

Run continuously, checking every interval:

```bash
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py --loop
```

Analyze scalp patterns and performance:

```bash
python3 bot_perpetual_scalper.py --analyze
```

## Environment Variables

### Market Configuration
- `SYMBOL` (default: `BNBUSDT`) - Trading pair symbol
- `INTERVAL` (default: `1m`) - Candle interval: 1m, 3m, 5m, 15m, 30m, 1h
- `NETWORK_TIMEOUT` (default: `5`) - Request timeout in seconds
- `MAX_RETRIES` (default: `3`) - Network retry attempts
- `RETRY_DELAY` (default: `1`) - Delay between retries in seconds

### Scalping Parameters
- `POSITION_SIZE_USDT` (default: `50`) - Base position size in USDT
- `SCALP_TARGET_BPS` (default: `10`) - Target profit in basis points
- `SCALP_STOP_BPS` (default: `5`) - Stop-loss in basis points
- `VOLATILITY_THRESHOLD` (default: `0.8`) - Minimum volatility to trade (0-1)
- `RSI_OVERSOLD` (default: `30`) - RSI threshold for long scalps
- `RSI_OVERBOUGHT` (default: `70`) - RSI threshold for short scalps
- `MAX_CONCURRENT_SCALPS` (default: `3`) - Maximum simultaneous positions
- `LEVERAGE` (default: `5`) - Leverage multiplier for position sizing

### Data Storage
- `SCALP_FILE` - Path to scalp trade history CSV
- `SCALP_PATTERN_FILE` - Path to scalp pattern analysis JSON
- `LEDGER_FILE` - Path to investment ledger (JSONL format)

## Scalping Logic

### Entry Signals

**Long Scalp** (oversold conditions):
- RSI drops below `RSI_OVERSOLD` threshold (default: 30)
- Current price trading above 50-period EMA (uptrend)
- Volatility exceeds `VOLATILITY_THRESHOLD`

**Short Scalp** (overbought conditions):
- RSI rises above `RSI_OVERBOUGHT` threshold (default: 70)
- Current price trading below 50-period EMA (downtrend)
- Volatility exceeds `VOLATILITY_THRESHOLD`

### Exit Logic

Each scalp has two exit levels:

1. **Target**: `SCALP_TARGET_BPS` basis points profit
2. **Stop-Loss**: `SCALP_STOP_BPS` basis points loss

If both trigger in the same candle, **target takes priority**.

### Position Sizing

```
Position Size = (POSITION_SIZE_USDT * LEVERAGE) / Entry Price
```

## Pattern Analysis

Run `--analyze` to view scalp statistics:

```json
{
  "total_scalps": 42,
  "win_rate": 71.4,
  "winners": 30,
  "losers": 12,
  "avg_pnl": 2.35,
  "total_pnl": 98.7,
  "avg_duration_seconds": 180
}
```

Metrics:
- **win_rate**: Percentage of profitable scalps
- **avg_pnl**: Average profit/loss per scalp (USDT)
- **total_pnl**: Cumulative P&L across all scalps
- **avg_duration_seconds**: Average time in position

## Indicators

### RSI (14-period)
Detects overbought (>70) and oversold (<30) conditions. Scalping favors mean reversion when RSI extremes align with EMA divergence.

### ATR (14-period)
Measures average true range. Used to scale stop-loss and target levels dynamically. Higher volatility → wider levels.

### Volatility (20-period)
Calculated as standard deviation of log returns. Scalping requires minimum volatility to avoid false signals in ranging markets.

### EMA (50-period)
Identifies primary trend direction. Scalps favor the trend:
- Long scalps when price > EMA
- Short scalps when price < EMA

## Safety & Limitations

✅ **What it does:**
- Detects and records micro-trend opportunities
- Calculates hypothetical fills at support/resistance levels
- Tracks performance metrics for strategy optimization
- Operates exclusively on Binance Testnet

❌ **What it does NOT do:**
- Submit actual orders or require API credentials
- Guarantee profitable trades
- Account for spread, slippage, partial fills, or exchange filters
- Manage real leverage, funding rates, or margin

## Unit Tests

```bash
python3 -m unittest discover -s tests -v
```

Covers:
- Indicator calculations (RSI, ATR, volatility, EMA)
- Entry signal detection (long/short conditions)
- Exit logic (target vs. stop-loss priority)
- PnL calculation and position sizing
- Scalp persistence and pattern analysis

## Backtest Against Historical Data

For serious analysis, backtest this logic against independent historical OHLCV data:

```bash
python3 bot_perpetual_scalper.py --backtest historical_data.csv
```

Review results carefully. Scalping performance is highly sensitive to:
- Slippage and spread assumptions
- Risk/reward ratio (target BPS vs. stop BPS)
- Volatility regime changes
- Maximum concurrent positions during drawdowns

## Limitations & Disclaimers

- **Educational only**: This is a learning tool, not trading advice or a guarantee of profit.
- **Testnet only**: The bot will refuse to run unless `BINANCE_TESTNET=1`.
- **No real trading**: No API credentials, no order submission, no live leverage.
- **Simplified fill logic**: Assumes fills at exact target/stop prices; real markets have slippage.
- **No funding costs**: Perpetuals incur funding fees not modeled here.
- **Cannot ensure profit**: Markets are unpredictable. Past performance ≠ future results.

## Example Commands

```bash
# Single evaluation cycle
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py

# Monitor continuously (check every 1 minute)
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py --loop

# Analyze performance after running for a while
python3 bot_perpetual_scalper.py --analyze

# Verify ledger integrity
python3 bot_perpetual_scalper.py --verify-ledger

# Use custom parameters
SYMBOL=ETHUSDT SCALP_TARGET_BPS=15 SCALP_STOP_BPS=8 \
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py --loop
```

## Next Steps

1. **Backtest on historical data** to validate the strategy across different market regimes
2. **Optimize parameters** (RSI thresholds, target/stop ratios) based on pattern analysis
3. **Analyze by time-of-day** to identify periods of higher edge
4. **Compare across symbols** to find which pairs scalp best
5. **Add risk metrics** like Sharpe ratio, maximum drawdown, recovery factor
