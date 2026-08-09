# Pilar 1 — Reputational Hardening (Code Signing)

## Problema

Las detecciones heurísticas `!ml` de antivirus (principalmente Windows Defender y similares) generan fricción de instalación y daño reputacional. El enfoque actual de ajustar heurísticas es una carrera sin fin.

## Solución

Implementar firma de código con certificados de alta confianza para establecer una reputación legítima ante Microsoft SmartScreen, Apple Gatekeeper y distribuciones Linux.

---

## Windows — Certificado EV (Extended Validation)

### Requisitos
- Certificado EV Code Signing emitido por una CA autorizada por Microsoft (DigiCert, Sectigo, GlobalSign)
- Almacenado en un HSM (Hardware Security Module) o en el servicio de firma en la nube del proveedor (ej. DigiCert KeyLocker)

### Integración en CI/CD (`_build.yml`)

```yaml
- name: Sign Windows binary (EV)
  env:
    DIGICERT_API_KEY: ${{ secrets.DIGICERT_API_KEY }}
    CERTIFICATE_FINGERPRINT: ${{ secrets.EV_CERT_FINGERPRINT }}
  run: |
    # Usar signtool.exe con el certificado EV via KeyLocker
    signtool sign \
      /tr http://timestamp.digicert.com \
      /td sha256 \
      /fd sha256 \
      /sha1 "$CERTIFICATE_FINGERPRINT" \
      codebase-memory-mcp.exe
```

### Reemplaza
El paso `codesign --sign -` (ad-hoc, sin identidad) actual en builds de macOS no tiene equivalente directo en Windows — actualmente Windows no firma. Este pilar lo introduce.

---

## macOS — Notarización con Apple Notary Service

### Requisitos
- Apple Developer ID Application certificate
- Cuenta Apple Developer activa
- `xcrun notarytool` (incluido en Xcode)

### Integración en CI/CD

```yaml
- name: Sign macOS binary (Developer ID)
  run: |
    codesign --sign "${{ secrets.APPLE_DEVELOPER_ID }}" \
             --options runtime \
             --timestamp \
             --force \
             build/c/codebase-memory-mcp

- name: Notarize macOS binary
  env:
    APPLE_ID: ${{ secrets.APPLE_ID }}
    APPLE_TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
    APPLE_APP_PASSWORD: ${{ secrets.APPLE_APP_PASSWORD }}
  run: |
    xcrun notarytool submit codebase-memory-mcp-darwin-*.tar.gz \
      --apple-id "$APPLE_ID" \
      --team-id "$APPLE_TEAM_ID" \
      --password "$APPLE_APP_PASSWORD" \
      --wait
```

### Reemplaza
El paso actual `codesign --sign - --force` (firma ad-hoc sin identidad verificada).

---

## Linux — Firma GPG de archivos de release

### Requisitos
- Par de claves GPG del proyecto (generadas una vez, clave privada en GitHub Secrets)
- Clave pública publicada en el repositorio (`PUBLIC_KEY.asc`) y en un keyserver

### Integración en CI/CD

```yaml
- name: Import GPG signing key
  run: echo "${{ secrets.GPG_PRIVATE_KEY }}" | gpg --import

- name: Sign Linux release archives
  run: |
    for archive in codebase-memory-mcp-linux-*.tar.gz; do
      gpg --batch --yes --armor \
          --detach-sign \
          --local-user "${{ secrets.GPG_KEY_ID }}" \
          "$archive"
    done

- name: Upload signatures as release assets
  # Subir los archivos .asc junto a los .tar.gz en el release de GitHub
```

### Verificación por el usuario
```bash
gpg --import PUBLIC_KEY.asc
gpg --verify codebase-memory-mcp-linux-amd64-portable.tar.gz.asc \
             codebase-memory-mcp-linux-amd64-portable.tar.gz
```

---

## Secrets requeridos en GitHub

| Secret | Descripción |
|--------|-------------|
| `DIGICERT_API_KEY` | API key de DigiCert KeyLocker (Windows EV) |
| `EV_CERT_FINGERPRINT` | SHA1 fingerprint del certificado EV |
| `APPLE_DEVELOPER_ID` | Identidad del Developer ID Application certificate |
| `APPLE_ID` | Apple ID para notarización |
| `APPLE_TEAM_ID` | Team ID de Apple Developer |
| `APPLE_APP_PASSWORD` | App-specific password para notarytool |
| `GPG_PRIVATE_KEY` | Clave privada GPG del proyecto (armored) |
| `GPG_KEY_ID` | ID de la clave GPG |

---

## Resultado esperado

- Windows SmartScreen: sin advertencias en instalación
- macOS Gatekeeper: binario verificado, sin cuarentena
- Linux: usuarios pueden verificar integridad criptográficamente
- Detecciones AV `!ml`: eliminadas permanentemente por reputación establecida
