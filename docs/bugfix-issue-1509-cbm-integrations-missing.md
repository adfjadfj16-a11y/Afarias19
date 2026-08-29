# Bug Fix: install.ps1 requires cbm-integrations.json missing from v0.9.0 Windows archive

**Upstream issue:** [DeusData/codebase-memory-mcp#1509](https://github.com/DeusData/codebase-memory-mcp/issues/1509)

## Problema

`install.ps1` (rama `main`) valida el archivo ZIP de Windows contra una lista exacta de archivos permitidos:

```
codebase-memory-mcp.exe, cbm-integrations.json, LICENSE, install.ps1, THIRD_PARTY_NOTICES.md
```

La validación falla con el release actual (v0.9.0) porque `cbm-integrations.json` **no está incluido** dentro del archivo `codebase-memory-mcp-windows-amd64.zip`:

```
error: unsafe or incomplete release archive: archive must contain exactly one cbm-integrations.json
```

## Causa raíz

El script `install.ps1` en `main` se adelantó al proceso de release: asume que `cbm-integrations.json` se empaqueta junto al binario Windows, pero el pipeline de CI/CD de v0.9.0 no lo incluyó en el ZIP.

## Solución

**Opción A (recomendada): Incluir `cbm-integrations.json` en el archivo ZIP de Windows en el próximo release**

El pipeline de build debe agregar `cbm-integrations.json` al ZIP de Windows antes de publicar el release. Esto hace que `install.ps1` funcione sin cambios.

```yaml
# En el workflow de release (_build.yml o equivalente)
- name: Package Windows archive
  run: |
    zip codebase-memory-mcp-windows-amd64.zip \
      codebase-memory-mcp.exe \
      cbm-integrations.json \
      LICENSE \
      install.ps1 \
      THIRD_PARTY_NOTICES.md
```

**Opción B (parche temporal): Hacer `cbm-integrations.json` opcional en `install.ps1`**

Mover `cbm-integrations.json` fuera de `$WindowsArchiveNames` y tratarlo como un archivo opcional (similar a como se trata `$uiPackName` para el variant `ui`). Solo aplica si el archivo existe en el ZIP.

```powershell
# En install.ps1 — validación con archivo opcional
$RequiredArchiveNames = @($BinName, "LICENSE", "install.ps1", "THIRD_PARTY_NOTICES.md")
$OptionalArchiveNames = @("cbm-integrations.json")
```

El bloque de validación del ZIP debe reemplazarse por:

```powershell
# ANTES (validación estricta — falla con v0.9.0):
# $archiveNames = $zip.Entries | Select-Object -ExpandProperty Name
# $missing = $WindowsArchiveNames | Where-Object { $_ -notin $archiveNames }
# if ($missing) { throw "unsafe or incomplete release archive: archive must contain exactly one $($missing -join ', ')" }

# DESPUÉS (parche Opción B — cbm-integrations.json opcional):
$archiveNames = $zip.Entries | Select-Object -ExpandProperty Name

$missingRequired = $RequiredArchiveNames | Where-Object { $_ -notin $archiveNames }
if ($missingRequired) {
    throw "unsafe or incomplete release archive: missing required files: $($missingRequired -join ', ')"
}

$missingOptional = $OptionalArchiveNames | Where-Object { $_ -notin $archiveNames }
if ($missingOptional) {
    Write-Warning "Optional file(s) not found in archive (will be skipped): $($missingOptional -join ', ')"
}
```

> **Nota:** Este parche permite instalar en v0.9.0 sin romper la validación de seguridad del ZIP. Los archivos requeridos siguen siendo verificados estrictamente.

**Recomendación:** La Opción A es la correcta a largo plazo. La Opción B puede servir como parche en `main` mientras se prepara el próximo release que incluya el archivo.

## Pasos para reproducir

```powershell
curl -fsSL https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.ps1 -o install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
# → error: unsafe or incomplete release archive: archive must contain exactly one cbm-integrations.json
```

## Impacto

- **Plataforma afectada:** Windows (amd64 y arm64)
- **Severidad:** Alta — el instalador principal de Windows no funciona en absoluto con v0.9.0
- **Workaround disponible:** Ninguno sin modificar el script manualmente

## Relación con el Roadmap

Este bug está relacionado con el **Pilar 1 (Code Signing)** y la estabilidad de la cadena de distribución. Un instalador roto en Windows daña directamente la confianza y la adopción. Se recomienda incluir la corrección en el próximo patch release antes de continuar con los pillares del roadmap.
