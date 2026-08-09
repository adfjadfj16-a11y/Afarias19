# Afarias19 Product Blueprint

## 1. Tipo de software

- **Producto:** SaaS propio de atención al cliente con asistencia por IA
- **Usuarios objetivo:** pequeños y medianos negocios
- **Problema a resolver:** captar prospectos, responder consultas iniciales y convertir pruebas gratis en clientes de pago con una experiencia simple y segura

## 2. Investigación de mercado legal

Se revisaron funciones y precios públicos de plataformas de soporte conocidas para tomar ideas de producto sin copiar código:

| Plataforma | Lo más visible públicamente | Entrada pública aproximada |
| --- | --- | --- |
| Intercom | atención conversacional, automatización, fuerte enfoque en IA | desde ~USD 39/mes |
| Zendesk | ticketing omnicanal, automatización, reportes | desde ~USD 19/agente/mes |
| Freshdesk | onboarding simple, automatización accesible, portal de autoservicio | desde ~USD 15/agente/mes |
| HubSpot Service Hub | integración con CRM, entrada gratuita y upsell posterior | gratis / desde ~USD 20/agente/mes |

Fuentes públicas revisadas en agosto de 2026:
- https://www.intercom.com/learning-center/customer-service-platform-comparison-2026
- https://www.gleap.io/blog/best-customer-support-software
- https://www.deelo.ai/blog/best-helpdesk-software-saas-companies-2026

## 3. Propuesta diferencial

Afarias19 se posiciona como una propuesta propia con:

- **30 días gratis** y pago flexible desde **USD 1** a partir del segundo mes
- **método de pago voluntario** configurable entre transferencia, tarjeta o PayPal
- **arquitectura sin dependencias externas** para arrancar con control total
- **IA inicial local** para preguntas frecuentes
- **base segura** desde el MVP

## 4. Arquitectura del sistema

- `cmd/server`: arranque del servicio
- `internal/app/server.go`: endpoints, lógica del producto y middleware de seguridad
- `internal/app/index.html`: landing page embebida
- `data/leads.json`: persistencia local de prospectos

## 5. Seguridad implementada desde el inicio

- validación de correo y campos obligatorios
- validación estricta de `Content-Type` y rechazo de campos JSON desconocidos
- límites de tamaño de cuerpo HTTP
- headers defensivos (`CSP`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`)
- control administrativo mediante token
- comparación segura de token usando hash + comparación constante
- escritura atómica para persistencia
- limitación simple por IP para reducir abuso
- limitación específica para intentos reiterados sobre el endpoint administrativo

## 6. Uso de IA en el MVP

- endpoint `POST /api/assistant`
- respuestas iniciales orientadas a precio, seguridad, automatización y ventas
- enfoque de IA local/reglada para no exponer datos a terceros en esta primera versión

## 7. Base de datos y modelo de datos

Entidad principal implementada:

- **Lead**
  - `id`
  - `name`
  - `email`
  - `company`
  - `goal`
  - `plan`
  - `paymentMethod`
  - `createdAt`
  - `trialEndsAt`

Decisiones:
- guardar solo datos mínimos del prospecto
- no almacenar contraseñas ni secretos de usuarios en esta fase
- dejar autenticación completa y roles finos como expansión siguiente

## 8. MVP construido

Ya está implementado:

- landing page comercial
- formulario de alta a prueba gratuita
- API de planes
- API de registro de prospectos
- API de asistente
- endpoint administrativo protegido
- persistencia local
- captura de preferencia de pago voluntario

## 9. Validación de calidad y seguridad

Validaciones ejecutadas:

- `go test ./...`
- `go build ./...`
- revisión automática de código
- escaneo de secretos

## 10. Guardado e iteración en el repositorio

La base quedó lista para seguir en etapas:

1. autenticación real de usuarios
2. panel administrativo visual
3. integración con proveedor de pagos
4. base de datos relacional
5. analítica, monitoreo y auditoría completa
