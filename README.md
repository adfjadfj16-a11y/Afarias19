# Afarias19

Aplicación local privada y personalizable creada en Go.

## Descripción

Este proyecto se convirtió en una mini app independiente para guardar información de forma local y privada en un archivo JSON dentro de `.afarias19/`.

## Requisitos

- Go 1.24 o superior
- Terminal / consola

## Ejecutar

```bash
go run .
```

## Funcionalidades

- menú interactivo
- guardar nombre, email y ciudad
- añadir y ver notas
- almacenamiento persistente privado localmente
- limpieza de datos y reinicio seguro

## Archivos importantes

- `persistence_manager.go`: lógica de almacenamiento y menú
- `.afarias19/datos.json`: archivo privado generado al ejecutar la app
- `PRIVATE_SETUP.md`: explicación de uso privado

## Nota

La app guarda datos de forma local en tu equipo y no requiere conexión externa.
