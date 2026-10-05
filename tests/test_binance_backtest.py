import csv
import tempfile
import unittest
from pathlib import Path

from binance_backtest import backtest, load_candles_csv, walk_forward
from bot_spot_binance_safe import Candle, SLOW_EMA


def sample_candles(count=320):
    candles = []
    for index in range(count):
        price = 100 + index * 0.02 + 6 * __import__("math").sin(index / 9)
        candles.append(
            Candle(
                index * 300_000,
                price,
                price * 1.006,
                price * 0.994,
                price,
                index * 300_000 + 299_999,
            )
        )
    return candles


class BacktestTests(unittest.TestCase):
    def test_csv_loads_chronological_ohlc(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "candles.csv"
            with path.open("w", newline="", encoding="utf-8") as file:
                writer = csv.writer(file)
                writer.writerow(["timestamp", "open", "high", "low", "close"])
                writer.writerow([1_700_000_000_000, 10, 11, 9, 10.5])
                writer.writerow([1_700_000_300_000, 10.5, 12, 10, 11])
            result = load_candles_csv(path)
        self.assertEqual(len(result), 2)
        self.assertEqual(result[1].close, 11)

    def test_csv_rejects_out_of_order_rows(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "candles.csv"
            path.write_text(
                "timestamp,open,high,low,close\n2000,10,11,9,10\n1000,10,11,9,10\n",
                encoding="utf-8",
            )
            with self.assertRaisesRegex(ValueError, "strictly chronological"):
                load_candles_csv(path)

    def test_backtest_reports_friction_and_buy_hold_benchmark(self):
        candles = sample_candles()
        free = backtest(
            candles, SLOW_EMA, len(candles), commission=0, spread_bps=0, slippage_bps=0
        )
        costly = backtest(
            candles, SLOW_EMA, len(candles), commission=0.002, spread_bps=10, slippage_bps=10
        )
        self.assertIn("buy_and_hold_return_pct", costly)
        self.assertLess(costly["buy_and_hold_return_pct"], free["buy_and_hold_return_pct"])
        self.assertIn("max_drawdown_pct", costly)
        self.assertIn("annualized_volatility_pct", costly)

    def test_backtest_rejects_period_without_warmup(self):
        with self.assertRaisesRegex(ValueError, "warm-up"):
            backtest(sample_candles(60), 10, 60)

    def test_walk_forward_keeps_final_holdout_after_validation(self):
        candles = sample_candles()
        result = walk_forward(candles, folds=4)
        folds = result["walk_forward"]
        self.assertEqual(len(folds), 4)
        self.assertLess(result["train"]["end_index"], folds[0]["start_index"])
        self.assertEqual(folds[-1]["end_index"], result["final_holdout"]["start_index"])
        self.assertFalse(result["final_holdout"]["used_for_tuning"])
        self.assertEqual(result["final_holdout"]["end_index"], len(candles))

    def test_walk_forward_requires_enough_candles(self):
        with self.assertRaisesRegex(ValueError, "Insufficient candles"):
            walk_forward(sample_candles(60), folds=4)


if __name__ == "__main__":
    unittest.main()
