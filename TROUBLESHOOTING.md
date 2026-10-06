# Troubleshooting Guide

Guía de solución de problemas comunes en Afarias19.

---

## Self-Check (Pilar 4)

### Problema: "FAIL: Archivo de configuración no encontrado"

**Síntomas:**
```
❌ Config | Missing configuration file
```

**Causas posibles:**
1. Archivo `config.json` no creado en `~/.config/afarias19/`
2. Permisos insuficientes para leer el archivo

**Solución:**
```bash
# Crear directorio y archivo de configuración
mkdir -p ~/.config/afarias19
cat > ~/.config/afarias19/config.json << EOF
{
  "symbol": "PEPEUSDT",
  "interval": "5m",
  "quote_order_size": 5.0
}
EOF

# Verificar permisos
ls -la ~/.config/afarias19/config.json
chmod 644 ~/.config/afarias19/config.json

# Ejecutar auto-test
./afarias19-linux-amd64
```

---

### Problema: "WARN: Variables de entorno incompletas"

**Síntomas:**
```
⚠️  Environment | Missing: BINANCE_API_KEY, BINANCE_API_SECRET
```

**Causas posibles:**
1. Variables no configuradas en `.env`
2. Variables configuradas pero no exportadas en el shell

**Solución:**
```bash
# Crear archivo .env
cat > .env << EOF
BINANCE_API_KEY=your_api_key_here
BINANCE_API_SECRET=your_secret_here
BINANCE_TESTNET=1
ENABLE_LIVE_TRADING=NO
EOF

# Cargar variables en el shell actual
source .env

# O exportarlas permanentemente (agregar a ~/.bashrc o ~/.zshrc)
export BINANCE_API_KEY=your_api_key_here
export BINANCE_API_SECRET=your_secret_here

# Verificar que están set
env | grep BINANCE

# Ejecutar auto-test
./afarias19-linux-amd64
```

**Nota:** Nunca commitear `.env` — está en `.gitignore`

---

### Problema: "FAIL: Permisos insuficientes en directorios"

**Síntomas:**
```
❌ Permissions | Cannot write to /home/user/.config/afarias19
```

**Causas posibles:**
1. Directorios propiedad de otro usuario
2. Bits de permiso incorrectos

**Solución:**
```bash
# Verificar propiedad
ls -la ~/.config/afarias19/

# Si pertenece a otro usuario:
sudo chown -R $USER:$USER ~/.config/afarias19/

# Fijar permisos correctos
chmod 755 ~/.config/afarias19/
chmod 644 ~/.config/afarias19/config.json

# Ejecutar auto-test nuevamente
./afarias19-linux-amd64
```

---

## Bot Trading (bot_spot_binance_safe.py)

### Problema: "ModuleNotFoundError: No module named 'requests'"

**Síntomas:**
```
Traceback (most recent call last):
  File "bot_spot_binance_safe.py", line 1, in <module>
    import requests
ModuleNotFoundError: No module named 'requests'
```

**Solución:**
```bash
# Instalar dependencias
pip install requests

# O si usas requirements.txt (cuando esté creado)
pip install -r requirements.txt

# Verificar instalación
python3 -c "import requests; print(requests.__version__)"

# Ejecutar bot
python3 bot_spot_binance_safe.py
```

---

### Problema: "Connection error: Cannot connect to Binance API"

**Síntomas:**
```
ERROR: Connection refused (111)
ERROR: Can't connect to Binance testnet
```

**Causas posibles:**
1. API keys incorrectas o expiradas
2. IP bloqueada por Binance
3. Red sin acceso a Internet

**Solución:**
```bash
# 1. Verificar API keys
echo "API Key: ${BINANCE_API_KEY:0:10}***"
echo "Secret: ${BINANCE_API_SECRET:0:10}***"

# 2. Verificar conectividad a Binance
curl -s https://api.binance.com/api/v3/ping
# Debe retornar: {}

# 3. Probar con testnet (recomendado)
export BINANCE_TESTNET=1
python3 bot_spot_binance_safe.py

# 4. Si siguen problemas, revisar logs
python3 -c "import logging; logging.basicConfig(level=logging.DEBUG); exec(open('bot_spot_binance_safe.py').read())"
```

---

### Problema: "InsufficientBalance: Insufficient balance"

**Síntomas:**
```
ERROR: {"code":-2010,"msg":"Insufficient balance for requested action."}
```

**Causas posibles:**
1. Saldo insuficiente en cuenta
2. Moneda equivocada especificada

**Solución:**
```bash
# Verificar saldo de testnet
curl -s "https://testnet.binance.vision/api/v3/account" \
  -H "X-MBX-APIKEY: $BINANCE_API_KEY" | jq '.balances[] | select(.free != "0")'

# Depositar en testnet (faucet)
# Ver: https://testnet.binance.vision/

# Ajustar QUOTE_ORDER_SIZE si es necesario
export QUOTE_ORDER_SIZE=1.0  # Cantidad más pequeña
python3 bot_spot_binance_safe.py
```

---

## Docker

### Problema: "docker: command not found"

**Solución:**
```bash
# Instalar Docker (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install docker.io docker-compose

# O en macOS (con Homebrew)
brew install docker docker-compose

# O descargar desde: https://www.docker.com/products/docker-desktop

# Verificar instalación
docker --version
docker-compose --version
```

---

### Problema: "permission denied while trying to connect to Docker daemon"

**Síntomas:**
```
permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock
```

**Solución (Linux):**
```bash
# Agregar usuario al grupo docker
sudo usermod -aG docker $USER

# Activar cambios de grupo (sin logout)
newgrp docker

# Verificar
docker ps

# Si aún hay problemas, reiniciar servicio Docker
sudo systemctl restart docker
```

---

### Problema: "docker-compose up" falla con "Cannot find image"

**Síntomas:**
```
ERROR: pull access denied for afarias19, repository does not exist or may require 'docker login'
```

**Solución:**
```bash
# Primero, buildear la imagen
make docker-build
# O manualmente:
docker build -t afarias19:latest .

# Luego ejecutar
docker-compose up -d

# Verificar imágenes disponibles
docker images | grep afarias19
```

---

### Problema: "docker-compose down" no detiene contenedores

**Solución:**
```bash
# Forzar parada
docker-compose down -v --remove-orphans

# O manualmente
docker stop $(docker ps -q)
docker rm $(docker ps -aq)

# Ver estado
docker ps -a
```

---

## Tests

### Problema: "go test: command not found"

**Solución:**
```bash
# Instalar Go
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# O en macOS
brew install go

# Verificar
go version
```

---

### Problema: "pytest: command not found"

**Solución:**
```bash
# Instalar pytest
pip install pytest pytest-cov

# Ejecutar tests
pytest tests/ -v

# O con makefile
make test-python
```

---

### Problema: "Race detector warnings en tests Go"

**Síntomas:**
```
==================
WARNING: DATA RACE
==================
```

**Solución:**
```bash
# Ejecutar tests CON detector de race
go test -race ./...

# Ver código problematico
go test -race -v cmd/self-check/...

# Agregar sincronización (mutex, channels, etc.)
# Buscar condiciones de carrera en persistence_manager.go
```

---

## CI/CD

### Problema: "GitHub Actions workflow falló"

**Diagnóstico:**
1. Ir a: https://github.com/adfjadfj16-a11y/Afarias19/actions
2. Hacer click en el workflow fallido
3. Expandir paso que falló
4. Leer logs completamente

**Soluciones comunes:**

#### Lint fail
```bash
# Local: Ejecutar formatter
make format

# Commit y push
git add .
git commit -m "style: format code"
git push
```

#### Test fail
```bash
# Local: Ejecutar tests
make test

# Debuggear
go test -v ./...

# Fijar bug y reintentarlo
git push
```

#### Security scan (Trivy)
```bash
# Instalar Trivy localmente
curl https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh

# Escanear
trivy fs .

# Fijar vulnerabilidades
# (Actualizar dependencias, usar versiones seguras, etc.)
```

---

## Performance

### Problema: "Bot es lento, hace trades tardíamente"

**Diagnóstico:**
```bash
# Ejecutar con profiling (si está implementado)
python3 -m cProfile -s cumtime bot_spot_binance_safe.py

# Ver logs detallados
export LOG_LEVEL=debug
python3 bot_spot_binance_safe.py 2>&1 | head -100
```

**Optimizaciones:**
1. Reducir `INTERVAL` (p.ej., de 5m a 1m)
2. Usar testnet para pruebas
3. Reducir cantidad de símbolos monitoreados
4. Mejorar conectividad a Internet

---

## Seguridad

### Problema: "Advertencia de secreto detectada en commits"

**Síntomas:**
```
⚠️  Secret scanning alert: Potential credentials found
```

**Solución INMEDIATA:**
1. Revocar el secret en la plataforma correspondiente
2. No commitear secretos
3. Usar `.env` o variables de entorno
4. Usar GitHub Secrets para CI/CD

```bash
# Verificar qué se commiteó accidentalmente
git log -p --all -S "API_KEY" | head -50

# Limpiar historio (destructivo)
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch .env' \
  --prune-empty --tag-name-filter cat -- --all

git push --force
```

---

## Contáctanos

Si el problema persiste:

1. **Revisar documentación:**
   - [README.md](README.md)
   - [CONTRIBUTING.md](CONTRIBUTING.md)
   - [SETUP_SECRETS.md](SETUP_SECRETS.md)

2. **Abrir Issue en GitHub:**
   - https://github.com/adfjadfj16-a11y/Afarias19/issues

3. **Descripción del Issue:**
   - Versión: `afarias19 -version`
   - SO: `uname -a`
   - Logs completos
   - Pasos para reproducir

---

**Última actualización:** 2026-10-06
