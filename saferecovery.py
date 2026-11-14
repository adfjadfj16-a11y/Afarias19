import shutil
import datetime
from pathlib import Path

origen = Path.home() / "mi_proyecto"
destino_base = Path.home() / "backup"
destino_base.mkdir(exist_ok=True)

if not origen.exists():
    print("Crea primero la carpeta ~/mi_proyecto y pon archivos dentro")
else:
    fecha = datetime.datetime.now().strftime("%Y%m%d_%H%M")
    destino = destino_base / f"backup_{fecha}"
    shutil.copytree(origen, destino)
    print(f"Backup listo en: {destino}")
    print("Para recuperar, copia de vuelta con cp -r")
