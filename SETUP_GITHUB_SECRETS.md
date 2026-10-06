# GitHub Secrets Configuration for Afarias19

Guía paso a paso para configurar secrets en GitHub y activar code signing.

---

## Overview

Los siguientes secrets deben configurarse en GitHub para que funcione el workflow `release.yml`:

| Secret Name | Platform | Used For | Service |
|------------|----------|----------|---------|
| `DIGICERT_CLIENT_ID` | Windows | DigiCert EV Code Signing | DigiCert |
| `DIGICERT_CLIENT_SECRET` | Windows | DigiCert Authentication | DigiCert |
| `DIGICERT_CERTIFICATE_SHA1` | Windows | DigiCert Certificate ID | DigiCert |
| `APPLE_DEVELOPER_ID_P12` | macOS | Developer ID Certificate (P12) | Apple |
| `APPLE_DEVELOPER_ID_PASSWORD` | macOS | P12 Certificate Password | Apple |
| `APPLE_DEVELOPER_ID_NAME` | macOS | Developer Name (e.g., "John Doe") | Apple |
| `APPLE_TEAM_ID` | macOS | Apple Team ID | Apple |
| `APPLE_NOTARIZE_PASSWORD` | macOS | App-specific Password | Apple |
| `GPG_PRIVATE_KEY` | Linux | GPG Private Key for Signing | GPG |
| `GPG_PASSPHRASE` | Linux | GPG Key Passphrase | GPG |

---

## Step-by-Step Setup

### 1. Acceso a GitHub Secrets

1. Ir a: `https://github.com/adfjadfj16-a11y/Afarias19`
2. Click en **Settings** (parte superior derecha)
3. En sidebar izquierdo: **Secrets and variables → Actions**
4. Click **"New repository secret"** para agregar cada secret

---

### 2. Windows Code Signing (DigiCert)

#### Paso A: Obtener credenciales de DigiCert

1. Registrarse en DigiCert Code Signing: https://www.digicert.com/
2. Comprar certificado EV Code Signing
3. En DigiCert dashboard:
   - Obtener `Client ID`
   - Obtener `Client Secret`
   - Obtener `Certificate SHA1` (fingerprint del certificado)

#### Paso B: Agregar secrets a GitHub

En la página de Secrets, agregar:

**Secret 1: DIGICERT_CLIENT_ID**
```
Name: DIGICERT_CLIENT_ID
Value: [tu_client_id_de_digicert]
```

**Secret 2: DIGICERT_CLIENT_SECRET**
```
Name: DIGICERT_CLIENT_SECRET
Value: [tu_client_secret_de_digicert]
```

**Secret 3: DIGICERT_CERTIFICATE_SHA1**
```
Name: DIGICERT_CERTIFICATE_SHA1
Value: [SHA1_fingerprint_del_certificado]
```

#### Paso C: Verificar en CI/CD

El workflow `release.yml` usará:
```bash
signtool sign /fd sha256 \
  /tr http://timestamp.digicert.com \
  /td sha256 \
  /dlib "path/to/digicert/library" \
  afarias19-windows-amd64.exe
```

---

### 3. macOS Code Signing (Apple Developer ID)

#### Paso A: Crear certificado Developer ID

1. Ir a: https://developer.apple.com/account
2. Login con Apple ID
3. **Certificates, Identifiers & Profiles**
4. **Certificates** → Click **"+"**
5. Seleccionar **"Developer ID Application"**
6. Seguir asistente (proporcionar CSR)
7. Descargar certificado (`.cer`)

#### Paso B: Exportar a P12

```bash
# Abre Keychain Access
# Localiza el certificado "Developer ID Application: ..."
# Right-click → Export
# Guarda como: DeveloperID.p12
# Ingresa contraseña (será el APPLE_DEVELOPER_ID_PASSWORD)

# Codificar como base64 para GitHub secret
base64 DeveloperID.p12 > DeveloperID.p12.b64
cat DeveloperID.p12.b64
```

#### Paso C: Obtener App-Specific Password

1. Ir a: https://appleid.apple.com/account/manage
2. **Security** → **App-Specific Passwords**
3. **Generate password**
4. Label: "Afarias19 Release"
5. Copiar contraseña (será `APPLE_NOTARIZE_PASSWORD`)

#### Paso D: Agregar secrets a GitHub

**Secret 1: APPLE_DEVELOPER_ID_P12**
```
Name: APPLE_DEVELOPER_ID_P12
Value: [contenido_base64_del_p12]
```

**Secret 2: APPLE_DEVELOPER_ID_PASSWORD**
```
Name: APPLE_DEVELOPER_ID_PASSWORD
Value: [contraseña_del_p12]
```

**Secret 3: APPLE_DEVELOPER_ID_NAME**
```
Name: APPLE_DEVELOPER_ID_NAME
Value: John Doe (o tu nombre de Apple Developer)
```

**Secret 4: APPLE_TEAM_ID**
```
Name: APPLE_TEAM_ID
Value: [tu_team_id_de_apple, p.ej., ABC123XYZ]
```

**Secret 5: APPLE_NOTARIZE_PASSWORD**
```
Name: APPLE_NOTARIZE_PASSWORD
Value: [app_specific_password_generada]
```

#### Paso E: Verificar en CI/CD

El workflow usará:
```bash
codesign -s "Developer ID Application: John Doe" \
  --options=runtime \
  afarias19-darwin-amd64

notarytool submit afarias19-darwin-amd64.zip \
  --apple-id your@email.com \
  --team-id ABC123XYZ \
  --password [APPLE_NOTARIZE_PASSWORD]
```

---

### 4. Linux Code Signing (GPG)

#### Paso A: Generar clave GPG (si no existe)

```bash
# Generar nueva clave GPG
gpg --full-generate-key

# Seleccionar:
# - Kind: RSA (4096 bits)
# - Expiry: 3 years
# - Name: Afarias19 Release Bot
# - Email: releases@afarias19.local
# - Password: [secure_passphrase]
```

#### Paso B: Exportar clave privada

```bash
# Listar claves
gpg --list-secret-keys

# Exportar privada en ASCII
gpg --armor --export-secret-keys [KEY_ID] > private-key.asc

# Codificar como base64 para GitHub
base64 private-key.asc > private-key.asc.b64
cat private-key.asc.b64
```

#### Paso C: Agregar secrets a GitHub

**Secret 1: GPG_PRIVATE_KEY**
```
Name: GPG_PRIVATE_KEY
Value: [contenido_base64_de_private_key.asc]
```

**Secret 2: GPG_PASSPHRASE**
```
Name: GPG_PASSPHRASE
Value: [contraseña_gpg_que_elegiste]
```

#### Paso D: Verificar en CI/CD

El workflow usará:
```bash
# Importar clave
echo "$GPG_PRIVATE_KEY" | base64 -d | gpg --import

# Firmar
gpg --armor --detach-sig afarias19-linux-amd64

# Exportar clave pública para verificación
gpg --armor --export [KEY_ID] > afarias19-public.asc
```

---

## Verificación de Secrets

### 1. Confirmar que todos están configurados

```bash
curl -X GET \
  -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/repos/adfjadfj16-a11y/Afarias19/actions/secrets | jq '.secrets[].name'
```

Debería retornar:
```
DIGICERT_CLIENT_ID
DIGICERT_CLIENT_SECRET
DIGICERT_CERTIFICATE_SHA1
APPLE_DEVELOPER_ID_P12
APPLE_DEVELOPER_ID_PASSWORD
APPLE_DEVELOPER_ID_NAME
APPLE_TEAM_ID
APPLE_NOTARIZE_PASSWORD
GPG_PRIVATE_KEY
GPG_PASSPHRASE
```

### 2. Disparar workflow manualmente

```bash
# Crear un tag para testing
git tag v1.0.0-rc1
git push origin v1.0.0-rc1

# O disparar manual en GitHub:
# - Actions → release.yml → "Run workflow" → Branch: main
```

### 3. Verificar logs del workflow

1. Ir a: https://github.com/adfjadfj16-a11y/Afarias19/actions
2. Click en el workflow que está en progreso
3. Ver logs en tiempo real
4. Buscar "Signing" o errores de autenticación

---

## Troubleshooting

### Error: "DigiCert Authentication Failed"

**Causa:** Cliente ID/Secret incorrecto

**Solución:**
```bash
# Verificar en DigiCert dashboard
# - Copiar exactamente sin espacios
# - Verificar que certificado está activo (no expirado)
# - Regenerar si es necesario
```

### Error: "Invalid P12 Certificate"

**Causa:** Certificado no es válido o contraseña incorrecta

**Solución:**
```bash
# Probar localmente primero
openssl pkcs12 -in DeveloperID.p12 -nodes -passin pass:PASSWORD

# Si falla, re-exportar desde Keychain
```

### Error: "GPG Key Import Failed"

**Causa:** Base64 encoding inválido

**Solución:**
```bash
# Verificar que está bien codificado
base64 -d <<< "$GPG_PRIVATE_KEY" | gpg --import --verbose

# Si falla, re-exportar
gpg --armor --export-secret-keys > private.asc
base64 private.asc | head -c 100  # Verificar salida
```

### Error: "Notarization Timed Out"

**Causa:** macOS Notary Service lento

**Solución:**
```bash
# Notarytool puede tardar 1-10 minutos
# Aumentar timeout en release.yml:
sleep 60 && notarytool info [SUBMISSION_ID] \
  --apple-id your@email.com \
  --password [PASSWORD] \
  --team-id [TEAM_ID]
```

---

## Seguridad Recomendada

### Proteger Secrets

1. **Limite de acceso:**
   - Settings → Secrets → Mostrar solo en workflows específicos
   - No exponerlo en logs (`secrets.DIGICERT_CLIENT_SECRET` es automáticamente enmascarado)

2. **Rotación periódica:**
   - Cambiar certificados cada 1-2 años
   - Revocar claves GPG si se comprometen

3. **Auditoría:**
   - GitHub Log: Ver quién accedió a secrets
   - Comando: `git log --all --grep="Secret" --oneline`

4. **No commitear secrets:**
   - Verificar `.gitignore` incluye `.env`, `*.p12`, `*.asc`
   - Usar `git secrets` para prevenir commits accidentales

---

## Validación Post-Setup

Una vez que todos los secrets están configurados:

```bash
# 1. Crear tag de test
git tag v1.0.0-test
git push origin v1.0.0-test

# 2. Ver workflow en acción
# GitHub → Actions → release.yml → ver estado

# 3. Verificar artifacts
# Si éxito: Release creado con binarios firmados en:
# https://github.com/adfjadfj16-a11y/Afarias19/releases/tag/v1.0.0-test

# 4. Validar firma (ejemplo para Linux)
gpg --import afarias19-public.asc
gpg --verify afarias19-linux-amd64.asc afarias19-linux-amd64
# Debe retornar: "Good signature from Afarias19 Release Bot"
```

---

## Referencias

- DigiCert Code Signing: https://www.digicert.com/code-signing
- Apple Developer ID: https://developer.apple.com/account/resources/certificates/
- GPG Key Management: https://gnupg.org/gph/en/manual/x110.html
- GitHub Actions Secrets: https://docs.github.com/en/actions/security-guides/encrypted-secrets

---

## Checklist Final

- [ ] Todos 10 secrets agregados a GitHub
- [ ] DigiCert certificado vigente (no expirado)
- [ ] Apple Developer ID certificado válido
- [ ] macOS App-Specific Password generada
- [ ] GPG clave privada exportada y codificada
- [ ] `.gitignore` protege archivos sensibles
- [ ] Workflow `release.yml` testeado exitosamente
- [ ] Release v1.0.0-test creada y verificada
- [ ] Binarios descargables con firmas válidas

---

**Última actualización:** 2026-10-06
**Siguiente paso:** Crear release oficial v1.0.0
