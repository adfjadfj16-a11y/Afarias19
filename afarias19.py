import datetime
import random

frases = [
    "Siempre perseverante, enfasis y energia positiva.",
    "Hoy avanzo un paso, aunque sea pequeno.",
    "La constancia le gana a la motivacion del dia."
]

hoy = datetime.date.today()
print(f"Afarias19 - {hoy}")
print(random.choice(frases))
nombre = input("Que lograste hoy? ")
with open("log.txt", "a") as f:
    f.write(f"{hoy} - {nombre}\n")
print("Guardado en log.txt")
