import logging
import os
from collections import defaultdict, deque
from typing import Deque, Dict

from dotenv import load_dotenv
from openai import AsyncOpenAI
from telegram import Update
from telegram.ext import (
    Application,
    CommandHandler,
    ContextTypes,
    MessageHandler,
    filters,
)

load_dotenv()

logging.basicConfig(
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    level=logging.INFO,
)
logger = logging.getLogger(__name__)

TELEGRAM_BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN")
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY")
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")
SYSTEM_PROMPT = os.getenv(
    "SYSTEM_PROMPT",
    "Eres un asistente útil y claro que responde en español.",
)
MAX_HISTORY_MESSAGES = int(os.getenv("MAX_HISTORY_MESSAGES", "8"))

if not TELEGRAM_BOT_TOKEN:
    raise RuntimeError("Falta TELEGRAM_BOT_TOKEN en las variables de entorno.")

if not OPENAI_API_KEY:
    raise RuntimeError("Falta OPENAI_API_KEY en las variables de entorno.")

client = AsyncOpenAI(api_key=OPENAI_API_KEY)
user_histories: Dict[int, Deque[dict]] = defaultdict(
    lambda: deque(maxlen=max(MAX_HISTORY_MESSAGES, 0))
)


def build_messages(user_id: int, user_message: str) -> list[dict]:
    history = user_histories[user_id]
    messages = [{"role": "system", "content": SYSTEM_PROMPT}]
    messages.extend(list(history))
    messages.append({"role": "user", "content": user_message})
    return messages


async def start_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    await update.message.reply_text(
        "¡Hola! Soy tu bot con IA. Escríbeme cualquier pregunta y te responderé."
    )


async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    await update.message.reply_text(
        "Comandos disponibles:\n"
        "/start - Inicia el bot\n"
        "/help - Muestra esta ayuda\n\n"
        "También puedes enviarme mensajes directamente."
    )


async def ask_openai(messages: list[dict]) -> str:
    response = await client.chat.completions.create(
        model=OPENAI_MODEL,
        messages=messages,
    )
    text = (response.choices[0].message.content or "").strip()
    return text or "No pude generar una respuesta en este momento."


async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    if not update.message or not update.effective_user:
        return

    user_id = update.effective_user.id
    user_text = (update.message.text or "").strip()
    if not user_text:
        return

    try:
        messages = build_messages(user_id=user_id, user_message=user_text)
        answer = await ask_openai(messages)

        history = user_histories[user_id]
        history.append({"role": "user", "content": user_text})
        history.append({"role": "assistant", "content": answer})

        await update.message.reply_text(answer)
    except Exception as error:
        logger.exception("Error al procesar mensaje: %s", error)
        await update.message.reply_text(
            "Ocurrió un error al consultar la IA. Inténtalo nuevamente en unos segundos."
        )


async def error_handler(update: object, context: ContextTypes.DEFAULT_TYPE) -> None:
    logger.exception("Excepción no controlada: %s", context.error)


def main() -> None:
    application = Application.builder().token(TELEGRAM_BOT_TOKEN).build()

    application.add_handler(CommandHandler("start", start_command))
    application.add_handler(CommandHandler("help", help_command))
    application.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, handle_message))
    application.add_error_handler(error_handler)

    logger.info("Bot iniciado...")
    application.run_polling(allowed_updates=Update.ALL_TYPES)


if __name__ == "__main__":
    main()
