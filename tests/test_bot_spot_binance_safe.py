import tempfile
import unittest
from unittest.mock import patch
from pathlib import Path

from bot_spot_binance_safe import (
    Candle,
    Indicators,
    INTERVAL,
    INTERVAL_SECONDS,
    RetryableError,
    calculate_indicators,
    default_state,
    ema,
    load_state,
    main,
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

    def test_indicators_are_finite_for_rising_market(self):
        result = calculate_indicators(candles_from_closes([100 + i for i in range(60)]))
        self.assertGreater(result.fast_ema, result.slow_ema)
        self.assertEqual(result.rsi, 100)
        self.assertGreater(result.atr, 0)

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

    def test_state_round_trip_and_parent_creation(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "nested" / "state.json"
            state = {**default_state(), "last_candle_time": 123}
            save_state(state, path)
            self.assertEqual(load_state(path), state)

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

    def test_loop_retries_transient_failures_with_backoff_and_stops_cleanly(self):
        with (
            patch("sys.argv", ["bot_spot_binance_safe.py", "--loop"]),
            patch(
                "bot_spot_binance_safe.run_once",
                side_effect=[
                    RetryableError("temporary network failure"),
                    None,
                    RetryableError("temporary network failure"),
                    None,
                ],
            ) as run_once_mock,
            patch(
                "bot_spot_binance_safe.time.sleep",
                side_effect=[None, None, None, KeyboardInterrupt],
            ) as sleep_mock,
        ):
            self.assertEqual(main(), 0)

        self.assertEqual(run_once_mock.call_count, 4)
        self.assertEqual(
            [call.args[0] for call in sleep_mock.call_args_list],
            [5, INTERVAL_SECONDS[INTERVAL], 5, INTERVAL_SECONDS[INTERVAL]],
        )

    def test_loop_stops_on_nonretryable_configuration_error(self):
        with (
            patch("sys.argv", ["bot_spot_binance_safe.py", "--loop"]),
            patch(
                "bot_spot_binance_safe.run_once",
                side_effect=RuntimeError("invalid configuration"),
            ),
            patch("bot_spot_binance_safe.time.sleep") as sleep_mock,
        ):
            self.assertEqual(main(), 1)
        sleep_mock.assert_not_called()


if __name__ == "__main__":
    unittest.main()
