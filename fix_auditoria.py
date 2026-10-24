import pathlib, re

# 1. Crear .env si no existe
env_path = pathlib.Path(".env")
if not env_path.exists():
    env_path = pathlib.Path("bot/.env")
if not env_path.exists():
    env_path = pathlib.Path(".env")
    env_path.write_text("""TRADING_PAIR=BTCUSDT
TRADE_FRACTION=0.10
POLL_INTERVAL_SECONDS=60
STRATEGY=sma_cross
DRY_RUN=true
LIVE_TRADING_ENABLED=false
MAX_ORDER_QUOTE=100
MAX_DAILY_LOSS_QUOTE=50
LOG_LEVEL=INFO
""")
    print(f"CREADO {env_path} con TOPE 10%")

# 2. Parchear config.py - forzar 0.10
cfg = pathlib.Path("config.py")
if not cfg.exists():
    cfg = pathlib.Path("bot/config.py")
if cfg.exists():
    txt = cfg.read_text()
    txt = txt.replace("0 < self.TRADE_FRACTION <= 1", "0 < self.TRADE_FRACTION <= 0.10")
    # Agregar candado Chile si no está
    if "CHILE INTOCABLE" not in txt:
        txt = txt.replace(
            'if not 0 < self.TRADE_FRACTION <= 0.10:',
            'if "CLP" in self.TRADING_PAIR or "CHILE" in self.TRADING_PAIR:\n            raise ValueError("CHILE INTOCABLE BLOQUEADO")\n        if not 0 < self.TRADE_FRACTION <= 0.10:'
        )
    cfg.write_text(txt)
    print(f"PARCHEADO {cfg}")

# 3. Parchear place_order para usar fraction_efectiva en BUY y SELL
bot = pathlib.Path("bot/fraccional_ia_binance.py")
if bot.exists():
    t = bot.read_text()
    # Asegurar que SELL también use fraction_efectiva
    t = t.replace("qty = balance * fraction", "qty = balance * fraction_efectiva")
    t = t.replace("spend = balance * fraction", "spend = balance * fraction_efectiva")
    bot.write_text(t)
    print(f"PARCHEADO {bot} - SELL y BUY usan TOPE 10%")

print("=== LISTO ===")
