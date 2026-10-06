# Configuración de Secrets para GitHub Actions

Este documento describe cómo configurar los secrets requeridos para **Pilar 1 (Code Signing)** y otros workflows de CI/CD.

## 🔐 Secrets Requeridos

### Para Pilar 1: Code Signing

#### Windows — DigiCert EV Certificate

**Qué es:**
- Certificado Extended Validation para firma de código en Windows
- Válido para SmartScreen (Microsoft confía automáticamente)
- Requiere certificado de una CA autorizada (DigiCert, Sectigo, GlobalSign, etc.)

**Cómo obtener:**
1. Ir a [DigiCert CodeSign](https://www.digicert.com/signing/code-signing-tools)
2. Solicitar certificado EV Code Signing
3. Generar par de claves (RSA 2048+)
4. Almacenar clave privada en **DigiCert KeyLocker** (cloud HSM)

**Secrets a configurar:**

```yaml
DIGICERT_API_KEY
  Descripción: API key para acceder a DigiCert KeyLocker
  Cómo obtener: DigiCert Dashboard → API Keys → Crear nueva
  Valor: Copiar la API key completa (formato: UUID)
  Seguridad: Nunca compartir, revocar si se expone
  
EV_CERT_FINGERPRINT
  Descripción: SHA1 fingerprint del certificado EV
  Cómo obtener: DigiCert Dashboard → Certificates → Seleccionar cert → SHA1 hash
  Valor: String hexadecimal de 40 caracteres
  Ejemplo: "3a4bcd5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c"
```

**Cómo configurar en GitHub:**

1. Ir al repositorio → Settings → Secrets and variables → Actions
2. Hacer clic en "New repository secret"
3. Nombre: `DIGICERT_API_KEY`
4. Valor: [Pegar API key de DigiCert]
5. Click "Add secret"
6. Repetir para `EV_CERT_FINGERPRINT`

**Validar:**

```bash
# En .github/workflows/release.yml, se ejecutará automáticamente cuando se cree un tag

# Trigger manual:
git tag v1.0.0-test
git push origin v1.0.0-test
# Luego revisar GitHub Actions → release workflow
```

---

#### macOS — Apple Developer ID Certificate

**Qué es:**
- Certificado Developer ID Application para firmar binarios macOS
- Notarización opcional pero recomendada (valida eternamente)
- Requiere Apple Developer Account (activo)

**Cómo obtener:**

1. Ir a [Apple Developer](https://developer.apple.com/account/)
2. Certificates, Identifiers & Profiles → Certificates
3. Click "+" → "Developer ID Application"
4. Seguir asistente de creación
5. Descargar certificado (.cer)
6. En Keychain, exportar como PKCS#12 (.p12):
   ```bash
   # Keychain Access → Seleccionar cert → File → Export Items...
   # Guardar como .p12, establecer contraseña fuerte
   ```

**Secrets a configurar:**

```yaml
APPLE_DEVELOPER_ID
  Descripción: Identidad del Developer ID (CN del certificado)
  Cómo obtener: Keychain → Seleccionar cert → mostrar detalles → Common Name
  Ejemplo: "Developer ID Application: Your Company (XXXXXXXXXX)"
  
APPLE_CERTIFICATE_P12
  Descripción: Certificado en formato base64 (para secrets)
  Cómo obtener: 
    cat certificate.p12 | base64 > cert.b64
  Valor: Contenido completo del archivo base64
  
APPLE_CERTIFICATE_PASSWORD
  Descripción: Contraseña del certificado .p12
  Valor: Contraseña que estableciste al exportar
  
APPLE_ID
  Descripción: Apple ID para notarización
  Valor: Tu Apple ID (email)
  
APPLE_TEAM_ID
  Descripción: Team ID de Developer Account
  Cómo obtener: Developer Account → Membership → Team ID
  Ejemplo: "XXXXXXXXXX"
  
APPLE_APP_PASSWORD
  Descripción: App-specific password para notarytool
  Cómo obtener: appleid.apple.com → App Passwords → Generar una
  Nota: Generar contraseña específica para "xcrun notarytool"
```

**Cómo configurar en GitHub:**

```bash
# 1. Preparar certificado en base64
cat certificate.p12 | base64 > cert.b64

# 2. Ir a Settings → Secrets and variables → Actions
# 3. Agregar secretos según la tabla anterior
```

**Validar:**

```bash
# En .github/workflows/release.yml, se ejecutará automáticamente con tag

# Trigger manual:
git tag v1.0.0-test-macos
git push origin v1.0.0-test-macos
```

---

#### Linux — GPG Signing

**Qué es:**
- Firma GPG detachada para archivos (*.tar.gz → *.tar.gz.asc)
- Usuario puede verificar: `gpg --verify file.asc file`
- Basado en criptografía de clave pública, sin requisito corporativo

**Cómo obtener:**

```bash
# 1. Generar par de claves (si no existe)
gpg --full-generate-key
  # Seleccionar: RSA (default)
  # Tamaño: 4096 bits
  # Validez: 1 año mínimo
  # Nombre: Tu nombre (o proyecto)
  # Email: email@project.com
  # Contraseña: fuerte

# 2. Listar claves
gpg --list-keys
  # Notar el KEY_ID (ej: 0A1B2C3D4E5F6A7B)

# 3. Exportar clave privada (armored)
gpg --armor --export-secret-keys KEY_ID > private.asc

# 4. Exportar clave pública (para repositorio)
gpg --armor --export KEY_ID > public.asc
```

**Secrets a configurar:**

```yaml
GPG_PRIVATE_KEY
  Descripción: Clave privada GPG en formato armor
  Cómo obtener: gpg --armor --export-secret-keys KEY_ID
  Valor: Contenido completo del archivo private.asc (incluye PGP markers)
  Ejemplo comienza con: "-----BEGIN PGP PRIVATE KEY BLOCK-----"
  
GPG_KEY_ID
  Descripción: ID de la clave GPG
  Cómo obtener: gpg --list-keys → 16 caracteres hexadecimales
  Ejemplo: "0A1B2C3D4E5F6A7B"
```

**Cómo configurar en GitHub:**

```bash
# 1. Copiar clave privada
cat private.asc

# 2. Settings → Secrets and variables → Actions → New secret
# Nombre: GPG_PRIVATE_KEY
# Valor: [Pegar contenido completo]

# 3. Repetir para GPG_KEY_ID
```

**Publicar clave pública en repositorio:**

```bash
# Subir public.asc al repositorio
cp public.asc /home/runner/work/Afarias19/Afarias19/
git add public.asc
git commit -m "Add GPG public key for release verification"
git push
```

**Validar:**

```bash
# Los usuarios pueden verificar:
gpg --import public.asc
gpg --verify afarias19-linux-amd64.asc afarias19-linux-amd64
# Output: "Good signature from ..."
```

---

### Para CI/CD General

#### GitHub Token (automático)

```yaml
GITHUB_TOKEN
  Descripción: Token de GitHub para crear releases, uploads, etc.
  Nota: Generado automáticamente por GitHub Actions
  Uso: Acceso a API de GitHub sin credenciales
```

**No requiere configuración manual** — GitHub lo genera automáticamente en cada workflow.

---

## 🔒 Mejores Prácticas

### Seguridad

1. **Nunca compartir secrets** — Revoca si se exponen
2. **Usar secretos específicos** — No reutilizar uno para todo
3. **Rotación periódica** — Cambiar contraseñas cada 90 días (aprox.)
4. **Auditoría** — GitHub logs quién accedió a cada secret
5. **Limpieza** — Eliminar secrets de proyectos discontinuados

### Backup

```bash
# Guardar clave privada GPG en lugar seguro (no en git)
gpg --export-secret-keys > ~/.gnupg/private-backup.gpg
# Guardar en:
# - Gestor de contraseñas (1Password, Bitwarden, etc.)
# - Hardware security key
# - Safe deposit box (para certificados EV)
```

### Verificación de Secrets

Antes de cada release, verificar que los secrets sean válidos:

```bash
# Localmente (sin GitHub):
gpg --import public.asc
gpg --list-keys

# En GitHub Actions (automático):
# - CI/CD pipeline intenta firmar
# - Si falla, se muestra en logs
```

---

## 🚨 Troubleshooting

### DigiCert API Key inválida

**Error:** `401 Unauthorized`

**Solución:**
1. Verificar que la API key sea correcta (copiar de nuevo)
2. Verificar que no tenga espacios en blanco
3. Verificar que la clave no esté revocada (Dashboard de DigiCert)
4. Crear nueva API key si es necesario

### Apple Certificate expirado

**Error:** `Certificate has expired`

**Solución:**
1. Verificar fecha de expiración: Keychain → Certificate
2. Si expiró, generar nuevo certificado en Developer Account
3. Exportar nuevo .p12
4. Actualizar `APPLE_CERTIFICATE_P12` en GitHub Secrets

### GPG signature invalid

**Error:** `gpg: BAD signature from ...`

**Solución:**
1. Verificar que `GPG_PRIVATE_KEY` sea correcto (incluye headers PGP)
2. Verificar que `GPG_KEY_ID` coincida con la clave privada
3. Probar localmente:
   ```bash
   echo "$GPG_PRIVATE_KEY" | gpg --import
   gpg --list-secret-keys
   ```

---

## 📋 Checklist Pre-Release

Antes de crear un tag (que dispara code signing):

- [ ] DigiCert API key válida y con fondos
- [ ] Apple certificate no expirado
- [ ] GPG private key importada localmente
- [ ] Tests locales pasando: `go test ./...`, `pytest tests/`
- [ ] Lint limpio: `golangci-lint run`, `black --check`
- [ ] Versión actualizada en código (si aplica)
- [ ] CHANGELOG.md actualizado
- [ ] Tests en GitHub Actions pasando (main branch)

```bash
# Crear tag
git tag v1.0.0

# Push (dispara release workflow)
git push origin v1.0.0

# Monitor en GitHub Actions → Workflows → Release with Code Signing
```

---

**Última actualización:** 2026-10-06
