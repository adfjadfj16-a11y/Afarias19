# Multi-stage build para máxima seguridad y eficiencia

# Etapa 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Instalar dependencias de build
RUN apk add --no-cache git gcc musl-dev

# Copiar módulos y source
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build
ARG VERSION=1.0.0
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-X 'main.Version=${VERSION}' -w -s" \
    -o /tmp/afarias19 \
    ./cmd/self-check

# Etapa 2: Runtime (distroless para máxima seguridad)
FROM alpine:3.20

LABEL maintainer="Afarias19 Project"
LABEL description="Afarias19: Sistema de confianza para infraestructura"
LABEL version="1.0.0"

# Instalar certificados CA para HTTPS
RUN apk add --no-cache ca-certificates

# Crear usuario no-root para seguridad
RUN addgroup -g 1000 afarias19 && \
    adduser -D -u 1000 -G afarias19 afarias19

# Crear directorios
RUN mkdir -p /home/afarias19/.config/afarias19 && \
    chown -R afarias19:afarias19 /home/afarias19

WORKDIR /home/afarias19

# Copiar binario desde builder
COPY --from=builder --chown=afarias19:afarias19 /tmp/afarias19 /usr/local/bin/afarias19

# Usuario no-root
USER afarias19

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD /usr/local/bin/afarias19 -json || exit 1

# Entrypoint
ENTRYPOINT ["/usr/local/bin/afarias19"]
CMD []
