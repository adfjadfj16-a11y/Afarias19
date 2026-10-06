# Binance Spot paper bot

`bot_spot_binance_safe.py` is an educational paper-trading tool. It requests
public candles from the hard-coded Binance Spot Testnet host and records
simulated trades locally. It does not read API keys, authenticate, or submit
orders. Its signal rules are an explainable baseline, not a claim of optimal
performance or investment advice.

## Run

From the repository root, use Python 3.10 or newer:

```sh
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_spot_binance_safe.py
```

To check every configured candle interval:

```sh
BINANCE_TESTNET=1 ENABLE_LIVE_TRADING=NO python3 bot_spot_binance_safe.py --loop
```

In loop mode, temporary market-data and storage failures are retried with
exponential backoff (5 to 60 seconds); the delay resets after a successful
check. Press Ctrl-C to stop cleanly. Position state is saved locally after each
processed candle so the bot can resume after a restart. This does not restart a
process terminated by the operating system; use a service manager if automatic
process restarts are needed.

Defaults are `PEPEUSDT`, `5m`, a simulated 5 USDT quote amount, and a 0.1%
simulated fee on each side. `SYMBOL`, `INTERVAL`, `QUOTE_ORDER_SIZE`,
`PAPER_FEE_RATE`, `STATE_FILE`, and `CSV_FILE` can be overridden with
environment variables. State and CSV output should be kept local; they contain
no API credentials.

The script stops unless `BINANCE_TESTNET=1` and
`ENABLE_LIVE_TRADING` is exactly `NO` (case-insensitive). The latter is a
fail-closed guard; the script has no live-trading implementation or order API.

## Strategy and limitations

It evaluates closed candles only. A simulated entry requires EMA(20) above
EMA(50), RSI(14) between 50 and 68, and positive ATR(14). A position exits at a
2-ATR stop, a 3-ATR target, or when EMA(20) falls below EMA(50) / RSI falls
below 45. If a candle touches both levels, the simulator assumes the stop
happened first. Fees are simulated; spread, slippage, partial fills, and
exchange filters are not. Backtest this logic against independent historical
data and include realistic costs before drawing conclusions. It cannot ensure
profit or prevent losses in real markets.

Unit tests:

```sh
python3 -m unittest discover -s tests -v
```
