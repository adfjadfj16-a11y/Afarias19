import tempfile
import unittest
from pathlib import Path
import json

from bot_perpetual_scalper import (
    Candle,
    ScalpIndicators,
    ScalpOpportunity,
    CompletedScalp,
    calculate_rsi,
    calculate_atr,
    calculate_volatility,
    calculate_ema,
    analyze_scalp_indicators,
    detect_scalp_opportunity,
    process_scalp_exit,
    save_completed_scalp,
    analyze_scalp_patterns,
)


def create_test_candles(closes, base_time=1000000):
    """Create test candles from close prices."""
    return [
        Candle(
            open_time=base_time + i * 60000,
            open=close * 0.998,
            high=close * 1.002,
            low=close * 0.998,
            close=close,
            volume=100.0,
            close_time=base_time + i * 60000 + 59999,
        )
        for i, close in enumerate(closes)
    ]


class PerpetualScalperTests(unittest.TestCase):
    
    def test_rsi_calculation_for_rising_prices(self):
        prices = [100 + i for i in range(20)]
        rsi = calculate_rsi(prices, 14)
        self.assertEqual(rsi, 100.0)
    
    def test_rsi_calculation_for_flat_prices(self):
        prices = [100] * 20
        rsi = calculate_rsi(prices, 14)
        self.assertEqual(rsi, 50.0)
    
    def test_rsi_calculation_for_falling_prices(self):
        prices = [100 - i for i in range(20)]
        rsi = calculate_rsi(prices, 14)
        self.assertEqual(rsi, 0.0)
    
    def test_rsi_rejects_insufficient_data(self):
        with self.assertRaisesRegex(ValueError, "Not enough prices"):
            calculate_rsi([100, 101], 14)
    
    def test_atr_calculation(self):
        candles = create_test_candles([100 + i for i in range(20)])
        atr = calculate_atr(candles, 14)
        self.assertGreater(atr, 0)
        # ATR should be reasonable for linearly increasing candles
        self.assertLess(atr, 5.0)
    
    def test_atr_rejects_insufficient_candles(self):
        candles = create_test_candles([100] * 10)
        with self.assertRaisesRegex(ValueError, "Not enough candles"):
            calculate_atr(candles, 14)
    
    def test_volatility_calculation(self):
        candles = create_test_candles([100 + i * 0.5 for i in range(30)])
        volatility = calculate_volatility(candles, 20)
        self.assertGreater(volatility, 0)
    
    def test_ema_calculation(self):
        prices = [100 + i for i in range(60)]
        ema = calculate_ema(prices, 50)
        self.assertGreater(ema, 0)
        self.assertTrue(abs(ema - prices[-1]) < 50)
    
    def test_indicators_analysis_for_valid_candles(self):
        # Oversold setup: falling prices with low RSI
        candles = create_test_candles([100 - i * 0.5 for i in range(60)])
        indicators = analyze_scalp_indicators(candles)
        
        self.assertLess(indicators.rsi, 50)
        self.assertGreater(indicators.atr, 0)
        self.assertGreater(indicators.volatility, 0)
        self.assertIsInstance(indicators.current_price, float)
    
    def test_indicators_analysis_rejects_insufficient_candles(self):
        candles = create_test_candles([100] * 20)
        with self.assertRaisesRegex(ValueError, "Not enough candles"):
            analyze_scalp_indicators(candles)
    
    def test_long_scalp_detection_oversold(self):
        # Price falling, then bounce above EMA
        candles = create_test_candles([100 - i * 0.3 for i in range(40)] + [95] * 10)
        indicators = analyze_scalp_indicators(candles)
        
        # Manually create oversold indicator with high volatility
        from bot_perpetual_scalper import VOLATILITY_THRESHOLD
        oversold_indicators = ScalpIndicators(
            rsi=25.0,
            atr=indicators.atr,
            volatility=VOLATILITY_THRESHOLD + 0.1,  # Ensure above threshold
            trend_ema=90.0,
            current_price=95.0
        )
        
        opp = detect_scalp_opportunity(candles, oversold_indicators, [])
        self.assertIsNotNone(opp)
        self.assertEqual(opp.direction, "LONG")
    
    def test_short_scalp_detection_overbought(self):
        # Price rising, then pullback below EMA
        candles = create_test_candles([100 + i * 0.3 for i in range(40)] + [105] * 10)
        indicators = analyze_scalp_indicators(candles)
        
        # Manually create overbought indicator with high volatility
        from bot_perpetual_scalper import VOLATILITY_THRESHOLD
        overbought_indicators = ScalpIndicators(
            rsi=75.0,
            atr=indicators.atr,
            volatility=VOLATILITY_THRESHOLD + 0.1,  # Ensure above threshold
            trend_ema=108.0,
            current_price=105.0
        )
        
        opp = detect_scalp_opportunity(candles, overbought_indicators, [])
        self.assertIsNotNone(opp)
        self.assertEqual(opp.direction, "SHORT")
    
    def test_scalp_opportunity_properties(self):
        candles = create_test_candles([100 - i * 0.3 for i in range(50)])
        indicators = analyze_scalp_indicators(candles)
        
        from bot_perpetual_scalper import VOLATILITY_THRESHOLD
        oversold_indicators = ScalpIndicators(
            rsi=25.0,
            atr=indicators.atr,
            volatility=VOLATILITY_THRESHOLD + 0.1,
            trend_ema=90.0,
            current_price=95.0
        )
        
        opp = detect_scalp_opportunity(candles, oversold_indicators, [])
        self.assertIsNotNone(opp)
        self.assertGreater(opp.target_price, opp.entry_price)
        self.assertLess(opp.stop_price, opp.entry_price)
        self.assertGreater(opp.position_size, 0)
    
    def test_long_scalp_exit_at_target(self):
        scalp = {
            'direction': 'LONG',
            'entry_price': 100.0,
            'target_price': 100.5,
            'stop_price': 99.5,
            'entry_time': 1000000,
            'position_size': 10.0,
        }
        
        completed = process_scalp_exit(scalp, 100.6, 1000060000)
        self.assertIsNotNone(completed)
        self.assertEqual(completed.exit_reason, "TARGET")
        self.assertGreater(completed.pnl_usdt, 0)
    
    def test_long_scalp_exit_at_stop(self):
        scalp = {
            'direction': 'LONG',
            'entry_price': 100.0,
            'target_price': 100.5,
            'stop_price': 99.5,
            'entry_time': 1000000,
            'position_size': 10.0,
        }
        
        completed = process_scalp_exit(scalp, 99.4, 1000060000)
        self.assertIsNotNone(completed)
        self.assertEqual(completed.exit_reason, "STOP")
        self.assertLess(completed.pnl_usdt, 0)
    
    def test_short_scalp_exit_at_target(self):
        scalp = {
            'direction': 'SHORT',
            'entry_price': 100.0,
            'target_price': 99.5,
            'stop_price': 100.5,
            'entry_time': 1000000,
            'position_size': 10.0,
        }
        
        completed = process_scalp_exit(scalp, 99.4, 1000060000)
        self.assertIsNotNone(completed)
        self.assertEqual(completed.exit_reason, "TARGET")
        self.assertGreater(completed.pnl_usdt, 0)
    
    def test_scalp_no_exit_in_range(self):
        scalp = {
            'direction': 'LONG',
            'entry_price': 100.0,
            'target_price': 100.5,
            'stop_price': 99.5,
            'entry_time': 1000000,
            'position_size': 10.0,
        }
        
        completed = process_scalp_exit(scalp, 100.1, 1000060000)
        self.assertIsNone(completed)
    
    def test_save_and_analyze_scalps(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            scalp_file = Path(tmpdir) / "scalps.csv"
            
            # Save some scalps
            scalp1 = CompletedScalp(
                timestamp=1000000,
                symbol="BNBUSDT",
                direction="LONG",
                entry_time=1000000,
                entry_price=100.0,
                exit_time=1000060,
                exit_price=100.5,
                position_size=10.0,
                pnl_usdt=5.0,
                pnl_percent=0.5,
                duration_seconds=60,
                exit_reason="TARGET",
                target_hit=True,
            )
            
            scalp2 = CompletedScalp(
                timestamp=1001000,
                symbol="BNBUSDT",
                direction="SHORT",
                entry_time=1001000,
                entry_price=100.0,
                exit_time=1001060,
                exit_price=99.8,
                position_size=10.0,
                pnl_usdt=2.0,
                pnl_percent=0.2,
                duration_seconds=60,
                exit_reason="TARGET",
                target_hit=True,
            )
            
            save_completed_scalp(scalp1, scalp_file)
            save_completed_scalp(scalp2, scalp_file)
            
            # Analyze patterns
            analysis = analyze_scalp_patterns(scalp_file)
            
            self.assertEqual(analysis["total_scalps"], 2)
            self.assertEqual(analysis["winners"], 2)
            self.assertEqual(analysis["win_rate"], 100.0)
            self.assertEqual(analysis["total_pnl"], 7.0)
            self.assertEqual(analysis["avg_pnl"], 3.5)
    
    def test_scalp_pattern_analysis_empty_file(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            scalp_file = Path(tmpdir) / "nonexistent.csv"
            analysis = analyze_scalp_patterns(scalp_file)
            
            self.assertEqual(analysis["total_scalps"], 0)
            self.assertEqual(analysis["win_rate"], 0.0)


if __name__ == "__main__":
    unittest.main()
