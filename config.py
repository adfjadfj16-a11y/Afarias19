"""
config.py - AUDITORIA TOPE 10% Y CHILE INTOCABLE
"""
import os
from dotenv import load_dotenv
load_dotenv()

class Config:
    BINANCE_API_KEY: str = os.getenv("BINANCE_API_KEY", "")
    BINANCE_API_SECRET: str = os.getenv("BINANCE_API_SECRET", "")
    BINANCE_TESTNET: bool = os.getenv("BINANCE_TESTNET", "true").lower() in ("true","1","yes")
    TRADING_PAIR: str = os.getenv("TRADING_PAIR", "BTCUSDT")
    TRADE_FRACTION: float = float(os.getenv("TRADE_FRACTION", "0.10"))
    POLL_INTERVAL_SECONDS: int = int(os.getenv("POLL_INTERVAL_SECONDS", "60"))
    STRATEGY: str = os.getenv("STRATEGY", "sma_cross")
    OPENAI_API_KEY: str = os.getenv("OPENAI_API_KEY", "")
    DRY_RUN: bool = os.getenv("DRY_RUN", "true").lower() in ("true","1","yes")
    LIVE_TRADING_ENABLED: bool = os.getenv("LIVE_TRADING_ENABLED", "false").lower() in ("true","1","yes")
    MAX_ORDER_QUOTE: float = float(os.getenv("MAX_ORDER_QUOTE", "100"))
    MAX_DAILY_LOSS_QUOTE: float = float(os.getenv("MAX_DAILY_LOSS_QUOTE", "50"))
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")

    def validate(self) -> None:
        # En DRY_RUN no exigimos keys reales
        if not self.DRY_RUN:
            if not self.BINANCE_API_KEY or not self.BINANCE_API_SECRET:
                raise ValueError("BINANCE_API_KEY y BINANCE_API_SECRET obligatorios para LIVE")
        # === AUDITORIA ===
        if "CLP" in self.TRADING_PAIR or "CHILE" in self.TRADING_PAIR or "BCLP" in self.TRADING_PAIR:
            raise ValueError(f"CHILE INTOCABLE BLOQUEADO: {self.TRADING_PAIR}")
        if not 0 < self.TRADE_FRACTION <= 0.10:
            raise ValueError(f"TOPE 10% VIOLADO: {self.TRADE_FRACTION}")

config = Config()
