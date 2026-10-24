import logging
from datetime import datetime

log = logging.getLogger("guardian")

def notificar_bloqueo(motivo, fraction_efectiva, pair="BTCUSDT"):
    msg = f"[{datetime.now()}] BLOQUEO GUARDIAN: {motivo} | pair={pair} fraction={fraction_efectiva*100:.1f}% | SALDO_CHILE_INTOCABLE no tocado"
    log.warning(msg)
    print(msg)
    # aqui puedes agregar Telegram si quieres
    return msg

def notificar_orden_ok(side, pair, fraction_efectiva):
    msg = f"[{datetime.now()}] ORDEN OK: {side} {pair} con {fraction_efectiva*100:.1f}% - tope 10% respetado"
    log.info(msg)
    print(msg)
