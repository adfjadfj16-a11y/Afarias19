# Configuración privada local

Este proyecto usa un almacenamiento local protegido en la carpeta `.afarias19/` para evitar exponer datos en un archivo visible del proyecto.

## Qué hace

- crea el directorio padre si no existe
- usa un archivo JSON privado en `.afarias19/datos.json`
- evita archivos vacíos y carga almacenamiento vacío como un mapa limpio
- escribe con permisos `600` para reducir exposición local
- usa una escritura segura con archivo temporal y renombrado atómico

## Uso

Ejecuta el programa con:

```bash
go run .
```

Si quieres cambiar la ruta de almacenamiento, modifica la variable `archivoPrivado` dentro de `persistence_manager.go`.

## Recomendación final

Para que el proyecto quede privado y local, mantén la carpeta `.afarias19/` fuera de repositorios compartidos y revisa el archivo `.gitignore` para no subir datos sensibles.
