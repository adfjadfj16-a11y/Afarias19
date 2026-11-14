import datetime
import json
from pathlib import Path

archivo = Path("enfasis.json")
if archivo.exists():
    datos = json.loads(archivo.read_text())
else:
    datos = {"dias": 0, "ultimo": ""}

hoy = str(datetime.date.today())
print("TOP ENFASIS Y PERSEVERANCIA")
print("Hoy es:", hoy)
print(f"Llevas {datos['dias']} dias registrados")

enfasis = input("Cual es tu enfasis de hoy? ")
accion = input("Que vas a perseverar hoy, aunque sea un paso pequeno? ")

datos["dias"] = datos["dias"] + 1
datos["ultimo"] = hoy
archivo.write_text(json.dumps(datos, indent=2))

with open("log.txt", "a") as f:
    f.write(f"{hoy} - Enfasis: {enfasis} - Perseverancia: {accion}\n")

print("Guardado. Siempre perseverante, enfasis y energia positiva.")
print(f"Total de dias: {datos['dias']}")
