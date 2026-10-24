import time, datetime, subprocess, os

LOG_FILE = os.path.expanduser("~/Afarias19/guardian.log")

def log_local(msg):
    ts = datetime.datetime.now().strftime("%H:%M:%S")
    line = f"[{ts}] {msg}"
    print(line)
    with open(LOG_FILE, "a") as f:
        f.write(line + "\n")

def vibrar():
    try:
        subprocess.run(["termux-vibrate", "-d", "400"], check=False, timeout=3)
    except: pass

def notificar(titulo, contenido):
    try:
        subprocess.run(["termux-notification", "--title", titulo, "--content", contenido], check=False, timeout=3)
    except: pass

def health_check():
    try:
        r = subprocess.run(["ping", "-c", "1", "-W", "2", "8.8.8.8"], capture_output=True, timeout=5)
        return r.returncode == 0
    except:
        return False

def ejecutar_seguro():
    if not health_check():
        log_local("Sin red -> backoff 30s (no destructivo)")
        vibrar()
        time.sleep(30)
        return
    log_local("[READ-ONLY] Check fraccional IA OK")
    notificar("Afarias19 Guardian OK", "Check read-only sin riesgo")

if __name__ == "__main__":
    ejecutar_seguro()
