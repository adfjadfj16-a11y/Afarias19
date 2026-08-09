# Afarias19

Afarias19 ahora incluye un MVP de **software SaaS de atención con IA** pensado para negocios pequeños que quieren atender mejor, empezar con **30 días gratis** y cobrar desde el segundo mes.

## Producto elegido

- **Tipo de software:** plataforma de atención al cliente con asistencia por IA
- **Público objetivo:** negocios pequeños y medianos
- **Problema principal:** responder rápido, capturar prospectos y ofrecer soporte inicial sin exponer datos sensibles

## Propuesta diferencial

- **30 días gratis** y cobro mensual desde el segundo mes
- **Asistente de IA local** para respuestas iniciales sin depender de servicios externos
- **Persistencia segura** de prospectos en JSON con escritura atómica
- **Seguridad básica desde el inicio**: validación de entradas, límites de tamaño, headers defensivos y token para endpoint administrativo

Más detalle en `docs/product_blueprint.md`.

## Arquitectura

- `cmd/server`: arranque del servidor HTTP
- `internal/app`: lógica de negocio, endpoints, almacenamiento y landing page embebida

## Endpoints principales

- `GET /` landing page
- `POST /signup` registro desde formulario web
- `GET /api/healthz` estado del servicio
- `GET /api/plans` detalle del plan gratis y plan de pago
- `POST /api/leads` registro de prospectos vía JSON
- `POST /api/assistant` respuestas iniciales del asistente
- `GET /api/admin/leads` listado administrativo protegido por token

## Variables de entorno

- `PORT`: puerto HTTP (por defecto `8080`)
- `DATA_FILE`: ruta del archivo de almacenamiento (por defecto `data/leads.json`)
- `ADMIN_TOKEN`: token requerido para `GET /api/admin/leads`

## Ejecutar localmente

```bash
cd <project-root>
go run ./cmd/server
```

Abrir `http://localhost:8080`.

## Probar

```bash
cd <project-root>
go test ./...
go build ./...
```

## Ejemplo de registro vía API

```bash
curl -X POST http://localhost:8080/api/leads \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"Ana Pérez",
    "email":"ana@example.com",
    "company":"Comercial Norte",
    "goal":"automatizar atención inicial"
  }'
```
