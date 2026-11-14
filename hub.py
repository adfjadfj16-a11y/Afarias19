import json
from pathlib import Path

archivo = Path("inversiones.json")
if archivo.exists():
    try:
        datos = json.loads(archivo.read_text())
    except Exception:
        datos = []
else:
    datos = []

print("Hub de inversiones, solo registro y simulacion")
print("No mueve dinero real, no promete ganancias.")
nombre = input("Nombre de la inversion de prueba: ")
try:
    invertido = float(input("Monto invertido de prueba, por ejemplo 100000: "))
    actual = float(input("Valor actual de prueba, por ejemplo 110000: "))
except ValueError:
    print("Escribe solo numeros. Intenta de nuevo.")
    invertido = 0
    actual = 0

if invertido > 0:
    roi = (actual - invertido) / invertido * 100
    datos.append({"nombre": nombre, "invertido": invertido, "actual": actual, "roi": roi})
    archivo.write_text(json.dumps(datos, indent=2))
    print(f"Guardado. ROI: {roi:.1f}%")
    total_inv = sum(d["invertido"] for d in datos)
    total_act = sum(d["actual"] for d in datos)
    roi_total = (total_act - total_inv) / total_inv * 100 if total_inv else 0
    print(f"Total registrado: invertido {total_inv}, actual {total_act}, ROI total {roi_total:.1f}%")
else:
    print("No se guardo porque el monto no era valido.")
