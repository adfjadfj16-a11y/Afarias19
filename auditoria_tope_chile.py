import re

ruta = "bot/fraccional_ia_binance.py"
with open(ruta, 'r', errors='ignore') as f:
    contenido = f.read()

print("=== AUDITORIA TOPE 10% Y CHILE INTOCABLE ===")
print(f"Archivo: {ruta}")

# Buscar tope
if re.search(r"0\.10|10%|tope", contenido, re.I):
    print("[OK] Menciona tope 10%")
else:
    print("[ALERTA] No encuentro lógica de tope 10%")

# Buscar Chile intocable
if re.search(r"CHILE|SALDO_CHILE", contenido, re.I):
    print("[REVISAR] Menciona CHILE - verifica que sea solo lectura, no trading")
    # muestra las lineas donde aparece
    for i, linea in enumerate(contenido.splitlines(), 1):
        if re.search(r"chile", linea, re.I):
            print(f"  L{i}: {linea.strip()}")
else:
    print("[OK] No toca Chile directamente")

# Buscar si hay proteccion de saldo
if "SALDO_CHILE_INTOCABLE" in contenido:
    print("[OK] Ya tiene variable SALDO_CHILE_INTOCABLE")
else:
    print("[FALTA] Agrega SALDO_CHILE_INTOCABLE y nunca lo uses en ordenes")
