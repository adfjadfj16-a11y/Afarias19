# Bot de Telegram con IA (Python + OpenAI)

Proyecto base para un bot de Telegram en Python que responde con la API de OpenAI.

## Requisitos

- Python 3.10+
- Token de bot de Telegram
- API key de OpenAI

## Configuración

1. Instala dependencias:

```bash
pip install -r requirements.txt
```

2. Copia el archivo de entorno y completa tus credenciales:

```bash
cp .env.example .env
```

Variables principales:

- `TELEGRAM_BOT_TOKEN`: token del bot en Telegram.
- `OPENAI_API_KEY`: clave API de OpenAI.
- `OPENAI_MODEL`: modelo de OpenAI (por defecto `gpt-4o-mini`).
- `SYSTEM_PROMPT`: prompt de sistema para el comportamiento del asistente.
- `MAX_HISTORY_MESSAGES`: cantidad máxima de mensajes recientes almacenados por usuario (cada intercambio usuario/asistente consume 2 mensajes).

## Ejecución

```bash
python telegram_ai_bot.py
```

## Comandos

- `/start` inicia la conversación
- `/help` muestra ayuda

## Notas

- El bot usa **polling** para simplificar el arranque inicial.
- El historial reciente se mantiene en memoria por usuario (no persistente).
