"""
config.py — Carga la configuración del bot desde variables de entorno (.env).
"""
import os
from dotenv import load_dotenv

load_dotenv()


class Config:
    # Binance
    BINANCE_API_KEY: str = os.getenv("BINANCE_API_KEY", "")
    BINANCE_API_SECRET: str = os.getenv("BINANCE_API_SECRET", "")
    BINANCE_TESTNET: bool = os.getenv("BINANCE_TESTNET", "true").lower() == "true"

    # Trading
    TRADING_PAIR: str = os.getenv("TRADING_PAIR", "BTCUSDT")
    TRADE_FRACTION: float = float(os.getenv("TRADE_FRACTION", "0.01"))
    POLL_INTERVAL_SECONDS: int = int(os.getenv("POLL_INTERVAL_SECONDS", "60"))

    # Estrategia
    STRATEGY: str = os.getenv("STRATEGY", "sma_cross")
    OPENAI_API_KEY: str = os.getenv("OPENAI_API_KEY", "")

    # Logging
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")

    def validate(self) -> None:
        if not self.BINANCE_API_KEY or not self.BINANCE_API_SECRET:
            raise ValueError(
                "BINANCE_API_KEY y BINANCE_API_SECRET son obligatorios. "
                "Copia .env.example como .env y rellena tus credenciales."
            )
        if not 0 < self.TRADE_FRACTION <= 1:
            raise ValueError("TRADE_FRACTION debe estar entre 0 y 1 (ej: 0.01 = 1%).")


config = Config()
