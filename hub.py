inversiones = [
    {"nombre": "Ejemplo 1", "invertido": 100000, "actual": 112000},
]
print("Investment Hub - Resumen")
for inv in inversiones:
    roi = (inv["actual"] - inv["invertido"]) / inv["invertido"] * 100
    print(f"{inv['nombre']}: ROI {roi:.1f}%")
