import json
from pathlib import Path

datos = [
    {"nombre": "Ejemplo 1", "invertido": 100000, "actual": 110000},
    {"nombre": "Ejemplo 2", "invertido": 50000, "actual": 48000},
]

print("Hub de inversiones, ejemplo de prueba")
print("No es dinero real, solo simulacion")
total_inv = 0
total_act = 0
for inv in datos:
    roi = (inv["actual"] - inv["invertido"]) / inv["invertido"] * 100
    print(f"{inv['nombre']}: invertido {inv['invertido']}, actual {inv['actual']}, ROI {roi:.1f}%")
    total_inv += inv["invertido"]
    total_act += inv["actual"]

roi_total = (total_act - total_inv) / total_inv * 100 if total_inv else 0
print(f"Total: invertido {total_inv}, actual {total_act}, ROI total {roi_total:.1f}%")

monto = 100000
for anio in range(1, 6):
    monto = monto * 1.05
    print(f"Ano {anio}: {int(monto)} con 5 por ciento anual de ejemplo")

Path("inversiones.json").write_text(json.dumps(datos, indent=2))
print("Guardado en inversiones.json")
