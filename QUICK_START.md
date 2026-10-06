# Quick Start Guide - Trading Bots

## Paper Trading Bot (Spot - Testnet)

Educational paper trading bot for PEPEUSDT spot market simulation.

```bash
# Single evaluation
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_spot_binance_safe.py

# Continuous monitoring
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_spot_binance_safe.py --loop

# View ledger integrity
python3 bot_spot_binance_safe.py --verify-ledger

# Custom parameters
SYMBOL=ADAUSDT QUOTE_ORDER_SIZE=10 PAPER_FEE_RATE=0.001 \
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_spot_binance_safe.py --loop
```

**Strategy**: EMA(20) > EMA(50) + RSI(14) [50, 68] + positive ATR
- Exit: Stop (2×ATR) or Target (3×ATR) or Signal reversal

**Output**: Trades logged to CSV + ledger


## Perpetual Scalping Bot (Futures - Testnet)

Scalping bot for USDT-M perpetuals with pattern analysis and risk management.

```bash
# Single cycle
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py

# Continuous scanning
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py --loop

# Analyze performance patterns
python3 bot_perpetual_scalper.py --analyze

# Custom parameters
SYMBOL=ETHUSDT INTERVAL=5m POSITION_SIZE_USDT=100 SCALP_TARGET_BPS=15 \
SCALP_STOP_BPS=8 MAX_CONCURRENT_SCALPS=5 LEVERAGE=3 \
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py --loop
```

**Strategy**: RSI extremes + EMA trend confirmation + volatility filter
- Entry: RSI < 30 (long) or RSI > 70 (short) + trend aligned + high vol
- Exit: Target (10 bps default) or Stop (5 bps default)
- Risk: Max 3 concurrent positions, dynamic levels per ATR

**Output**: Scalps logged to CSV + event ledger + pattern analysis


## Data Directories

All personal financial data stored outside repository:

```
~/.local/state/afarias19/binance/
├── position_spot_testnet.json      # Current spot position state
├── trades_spot_testnet.csv         # Spot trade history
├── scalps_perpetual.jsonl          # Individual scalp events
├── scalps_perpetual.csv            # Scalp trade summary
└── investment-ledger.jsonl         # Complete transaction ledger
```


## Environment Variables - Common

| Variable | Default | Purpose |
|----------|---------|---------|
| `BINANCE_TESTNET` | (required) | Set to "1" to enable testnet mode |
| `ENABLE_LIVE_TRADING` | "NO" | Safety guard: must be "NO" |
| `NETWORK_TIMEOUT` | 5s | Request timeout |
| `MAX_RETRIES` | 3 | Network retry attempts |

## Environment Variables - Spot Bot

| Variable | Default | Purpose |
|----------|---------|---------|
| `SYMBOL` | PEPEUSDT | Trading pair |
| `INTERVAL` | 5m | Candle interval |
| `QUOTE_ORDER_SIZE` | 5 USDT | Position size |
| `PAPER_FEE_RATE` | 0.1% | Simulated fee |
| `STATE_FILE` | (data dir) | State persistence path |
| `CSV_FILE` | (data dir) | Trade history path |
| `LEDGER_FILE` | (data dir) | Event ledger path |

## Environment Variables - Scalping Bot

| Variable | Default | Purpose |
|----------|---------|---------|
| `SYMBOL` | BNBUSDT | Trading pair |
| `INTERVAL` | 1m | Candle interval |
| `POSITION_SIZE_USDT` | 50 | Base position size |
| `SCALP_TARGET_BPS` | 10 | Target profit (basis points) |
| `SCALP_STOP_BPS` | 5 | Stop loss (basis points) |
| `VOLATILITY_THRESHOLD` | 0.8% | Min vol to trade |
| `RSI_OVERSOLD` | 30 | Long entry threshold |
| `RSI_OVERBOUGHT` | 70 | Short entry threshold |
| `MAX_CONCURRENT_SCALPS` | 3 | Max simultaneous positions |
| `LEVERAGE` | 5x | Position leverage |


## Testing

Run all tests:

```bash
python3 -m unittest discover -s tests -v
```

Spot bot tests:
```bash
python3 -m unittest tests.test_bot_spot_binance_safe -v
```

Scalping bot tests (19 tests):
```bash
python3 -m unittest tests.test_perpetual_scalper -v
```

Private ledger tests:
```bash
python3 -m unittest tests.test_private_ledger -v
```

Backtest utilities:
```bash
python3 -m unittest tests.test_binance_backtest -v
```


## Safety Checklist Before Running

- [ ] Set `BINANCE_TESTNET=1` (not production API)
- [ ] Set `ENABLE_LIVE_TRADING=NO` (safety guard)
- [ ] Verify API keys are NOT in environment
- [ ] Confirm no real credentials in state files
- [ ] Data dir is outside repository (auto-enforced)
- [ ] Understand you're using **testnet/paper only**
- [ ] Run tests first: `python3 -m unittest discover -s tests -v`


## Troubleshooting

### "Set BINANCE_TESTNET=1"
```bash
# Fix: Export the flag
export BINANCE_TESTNET=1
python3 bot_spot_binance_safe.py
```

### "ENABLE_LIVE_TRADING must remain NO"
```bash
# Never set this:
export ENABLE_LIVE_TRADING=YES  # ❌ WRONG

# Correct:
export ENABLE_LIVE_TRADING=NO   # ✅ Correct
```

### "Refused to store data inside repository"
The bots intentionally prevent storing financial data in the repo. Check that `STATE_FILE`, `CSV_FILE`, and `LEDGER_FILE` point to paths outside the repository, OR let them default to `~/.local/state/afarias19/binance/`.

### Connection timeouts
Increase `NETWORK_TIMEOUT`:
```bash
NETWORK_TIMEOUT=10 BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_spot_binance_safe.py
```

### Pattern analysis shows no data
Run the bot first to generate scalps:
```bash
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_perpetual_scalper.py --loop &
sleep 60
python3 bot_perpetual_scalper.py --analyze
```


## Performance Analysis

After running scalping bot, analyze results:

```bash
python3 bot_perpetual_scalper.py --analyze
```

Example output:
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

Metrics interpretation:
- **win_rate > 60%**: Strategy is catching good moves
- **avg_pnl > 1 USDT**: Profitable after costs
- **avg_duration < 300s**: True scalping (short holds)
- **total_pnl > 0**: Net profitability across sample


## Documentation

Full docs available in:
- `docs/binance-spot-paper-bot.md` - Spot bot details
- `docs/perpetual-scalper.md` - Scalping strategy & tuning

Source code:
- `bot_spot_binance_safe.py` - Spot bot (366 lines)
- `bot_perpetual_scalper.py` - Scalping bot (500+ lines)
- `private_ledger.py` - Ledger utilities
- `binance_backtest.py` - Backtest engine


---

**Remember**: These are educational testnet-only tools. They cannot profit or prevent losses in real markets. Use them to learn, backtest, and validate strategy ideas before any consideration of live trading.
