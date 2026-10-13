# ==============================================================================
# Multi-stage Dockerfile for Cobalt Core Monolith
# ==============================================================================

# ------------------------------------------------------------------------------
# Stage 1: Frontend Builder (React + Vite + TypeScript)
# ------------------------------------------------------------------------------
FROM node:20-alpine AS web-builder
WORKDIR /app/web

# Copy frontend source
COPY web/ ./

# Build frontend if package.json exists; otherwise generate fallback placeholder
RUN if [ -f package.json ]; then \
        npm ci --omit=dev || npm install; \
        npm run build; \
    else \
        mkdir -p dist && echo '<!DOCTYPE html><html><head><title>Cobalt Core</title></head><body><h1>Cobalt Core</h1></body></html>' > dist/index.html; \
    fi

# ------------------------------------------------------------------------------
# Stage 2: Backend Builder (Go Monolith)
# ------------------------------------------------------------------------------
FROM golang:1.23-alpine AS go-builder
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata git

# Cache dependencies
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy Go backend sources and compiled web assets
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist

# Compile static Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /cobalt ./cmd/server

# ------------------------------------------------------------------------------
# Stage 3: Minimal Production Runtime
# ------------------------------------------------------------------------------
FROM alpine:3.20 AS runtime
WORKDIR /app

# Install security certificates & timezone data, create unprivileged user
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S cobalt && adduser -S cobalt -G cobalt

# Copy binary from builder
COPY --from=go-builder /cobalt /app/cobalt

# Use non-root user
USER cobalt

# 8080: HTTP Web UI / API
# 1883: Embedded MQTT Broker
EXPOSE 8080 1883

ENTRYPOINT ["/app/cobalt"]
