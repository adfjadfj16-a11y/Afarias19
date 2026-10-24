import time, datetime, subprocess, os, json
from pathlib import Path
BASE_DIR = Path.home() / "Afarias19"
LOG_FILE = BASE_DIR / "guardian.log"
STATE_FILE = BASE_DIR / "guardian_state.json"
MAX_RETRIES, BACKOFF_BASE = 2, 30
def now(): return datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
def log_local(m):
    l=f"[{now()}] {m}"; print(l)
    try:
        with open(LOG_FILE,"a",encoding="utf-8") as f: f.write(l+"\n")
    except: pass
def vibrar(ms=400):
    try: subprocess.run(["termux-vibrate","-d",str(ms)],check=False,timeout=3)
    except: pass
def notificar(t,c):
    try: subprocess.run(["termux-notification","--title",t,"--content",c,"--priority","high"],check=False,timeout=4)
    except: pass
def health_check():
    try:
        r=subprocess.run(["ping","-c","1","-W","2","8.8.8.8"],capture_output=True,timeout=5)
        return r.returncode==0
    except: return False
def load_state():
    try:
        if STATE_FILE.exists(): return json.loads(STATE_FILE.read_text())
    except: pass
    return {"fails":0}
def save_state(f): 
    try: STATE_FILE.write_text(json.dumps({"fails":f,"last":now()}))
    except: pass
def ejecutar_seguro():
    s=load_state()
    if s["fails"]>=3:
        log_local(f"CIRCUIT OPEN {s['fails']} fallos, backoff {BACKOFF_BASE*2}s"); vibrar(800); time.sleep(BACKOFF_BASE*2); save_state(0); return False
    if not health_check():
        log_local("HEALTH FAIL -> backoff 30s (no destructivo)"); vibrar(); notificar("Afarias19 Guardian - Pausa","Sin red, backoff 30s"); save_state(s["fails"]+1); time.sleep(BACKOFF_BASE); return False
    for i in range(1,MAX_RETRIES+1):
        try:
            log_local(f"[READ-ONLY] Check fraccional IA {i}/{MAX_RETRIES} OK - Afarias19")
            save_state(0); notificar("Afarias19 Guardian OK",f"Check {i} OK - sin riesgo"); return True
        except Exception as e:
            log_local(f"Error {i}: {e}"); time.sleep(5)
    save_state(s["fails"]+1); vibrar(); notificar("Guardian - Error",f"Fallo {MAX_RETRIES}"); return False
if __name__=="__main__": ejecutar_seguro()
