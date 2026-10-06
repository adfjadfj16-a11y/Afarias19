import tempfile
import unittest
from unittest.mock import patch, MagicMock
from pathlib import Path
import json
import math

from bot_spot_binance_safe import (
    Candle,
    Indicators,
    calculate_indicators,
    default_state,
    ema,
    load_state,
    process_candle,
    run_once,
    save_state,
)


def candles_from_closes(closes):
    return [
        Candle(
            index * 300_000,
            close,
            close * 1.01,
            close * 0.99,
            close,
            index * 300_000 + 299_999,
        )
        for index, close in enumerate(closes)
    ]


class BinancePaperBotTests(unittest.TestCase):
    def test_ema_matches_hand_calculated_values(self):
        self.assertEqual(ema([1, 2, 3, 4], 2), 3.5)

    def test_ema_rejects_invalid_period(self):
        with self.assertRaisesRegex(ValueError, "period must be at least 1"):
            ema([1, 2, 3], 0)

    def test_ema_rejects_insufficient_values(self):
        with self.assertRaisesRegex(ValueError, "Not enough values"):
            ema([1, 2], 3)

    def test_ema_rejects_non_finite_values(self):
        with self.assertRaisesRegex(ValueError, "must all be finite"):
            ema([1, 2, float('inf'), 4], 2)

    def test_indicators_are_finite_for_rising_market(self):
        result = calculate_indicators(candles_from_closes([100 + i for i in range(60)]))
        self.assertGreater(result.fast_ema, result.slow_ema)
        self.assertEqual(result.rsi, 100)
        self.assertGreater(result.atr, 0)
        self.assertTrue(math.isfinite(result.fast_ema))
        self.assertTrue(math.isfinite(result.slow_ema))
        self.assertTrue(math.isfinite(result.rsi))
        self.assertTrue(math.isfinite(result.atr))

    def test_indicators_for_flat_market(self):
        result = calculate_indicators(candles_from_closes([100] * 60))
        self.assertEqual(result.fast_ema, result.slow_ema)
        self.assertEqual(result.rsi, 50.0)
        self.assertGreater(result.atr, 0)

    def test_indicators_reject_insufficient_candles(self):
        with self.assertRaisesRegex(ValueError, "Not enough candles"):
            calculate_indicators(candles_from_closes([100] * 10))

    def test_entry_is_paper_only_and_not_repeated_for_same_candle(self):
        candle = candles_from_closes([100] * 60)[-1]
        bullish = Indicators(fast_ema=102, slow_ema=101, rsi=55, atr=1)
        state, trade = process_candle(default_state(), candle, bullish)
        self.assertEqual(trade["action"], "PAPER_BUY")
        self.assertTrue(state["in_position"])
        duplicate_state, duplicate_trade = process_candle(state, candle, bullish)
        self.assertIsNone(duplicate_trade)
        self.assertEqual(duplicate_state, state)

    def test_stop_is_used_if_stop_and_target_hit_in_same_candle(self):
        candle = Candle(1, open=105, high=120, low=90, close=105, close_time=2)
        state = {
            **default_state(),
            "in_position": True,
            "quantity": 0.1,
            "quote_spent": 10,
            "stop_price": 95,
            "take_profit_price": 110,
        }
        indicators = Indicators(fast_ema=105, slow_ema=100, rsi=55, atr=1)
        new_state, trade = process_candle(state, candle, indicators)
        self.assertEqual(trade["action"], "PAPER_SELL_STOP")
        self.assertEqual(trade["price"], 95)
        self.assertFalse(new_state["in_position"])

    def test_stop_fill_accounts_for_gap_below_stop(self):
        candle = Candle(1, open=90, high=96, low=89, close=91, close_time=2)
        state = {
            **default_state(),
            "in_position": True,
            "quantity": 0.1,
            "quote_spent": 10,
            "stop_price": 95,
            "take_profit_price": 110,
        }
        indicators = Indicators(fast_ema=105, slow_ema=100, rsi=55, atr=1)
        _, trade = process_candle(state, candle, indicators)
        self.assertEqual(trade["price"], 90)

    def test_process_candle_rejects_invalid_exit_price(self):
        candle = Candle(1, open=-5, high=0, low=-10, close=-1, close_time=2)
        state = {
            **default_state(),
            "in_position": True,
            "quantity": 0.1,
            "quote_spent": 10,
            "stop_price": 95,
            "take_profit_price": 110,
        }
        indicators = Indicators(fast_ema=105, slow_ema=100, rsi=55, atr=1)
        with self.assertRaisesRegex(ValueError, "Invalid exit price"):
            process_candle(state, candle, indicators)

    def test_process_candle_rejects_invalid_entry_price(self):
        candle = Candle(1, open=-5, high=-1, low=-10, close=-1, close_time=2)
        bullish = Indicators(fast_ema=102, slow_ema=101, rsi=55, atr=1)
        with self.assertRaisesRegex(ValueError, "Invalid entry price"):
            process_candle(default_state(), candle, bullish)

    def test_state_round_trip_and_parent_creation(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "nested" / "state.json"
            state = {**default_state(), "last_candle_time": 123}
            save_state(state, path)
            self.assertEqual(load_state(path), state)

    def test_load_state_rejects_corrupted_json(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "state.json"
            path.write_text("{ invalid json }")
            with self.assertRaisesRegex(RuntimeError, "not valid JSON"):
                load_state(path)

    def test_load_state_rejects_mismatched_symbol(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "state.json"
            state = {**default_state(), "symbol": "DIFFERENTUSDT"}
            path.write_text(json.dumps(state))
            with self.assertRaisesRegex(RuntimeError, "belongs to"):
                load_state(path)

    def test_load_state_rejects_invalid_position_flag(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "state.json"
            state = {**default_state(), "in_position": "yes"}
            path.write_text(json.dumps(state))
            with self.assertRaisesRegex(RuntimeError, "Invalid position flag"):
                load_state(path)

    def test_load_state_rejects_negative_quantities(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "state.json"
            state = {**default_state(), "quantity": -1.0}
            path.write_text(json.dumps(state))
            with self.assertRaisesRegex(RuntimeError, "Invalid quantity"):
                load_state(path)

    def test_load_state_rejects_inconsistent_position(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "state.json"
            state = {
                **default_state(),
                "in_position": True,
                "quantity": 0.1,
                "entry_price": 100,
                "quote_spent": 10,
                "stop_price": 150,
                "take_profit_price": 90,
            }
            path.write_text(json.dumps(state))
            with self.assertRaisesRegex(RuntimeError, "stop_price < take_profit_price"):
                load_state(path)

    def test_live_trading_setting_fails_closed_before_network_access(self):
        with patch.dict(
            "os.environ",
            {"BINANCE_TESTNET": "1", "ENABLE_LIVE_TRADING": "YES"},
            clear=False,
        ):
            with self.assertRaisesRegex(RuntimeError, "must remain NO"):
                run_once()

    def test_missing_testnet_flag_fails_before_network_access(self):
        with patch.dict(
            "os.environ",
            {"BINANCE_TESTNET": "0", "ENABLE_LIVE_TRADING": "NO"},
            clear=False,
        ):
            with self.assertRaisesRegex(RuntimeError, "Only Binance Testnet"):
                run_once()


if __name__ == "__main__":
    unittest.main()
