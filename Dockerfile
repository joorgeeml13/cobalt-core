# Multi-stage Dockerfile for Cobalt Core monolith

# 1. Frontend build stage (React + Vite)
FROM node:20-alpine AS web-builder
WORKDIR /app/web

COPY web/ ./
RUN if [ -f package.json ]; then \
        npm ci --omit=dev || npm install; \
        npm run build; \
    else \
        mkdir -p dist && echo '<!DOCTYPE html><html><head><title>Cobalt Core</title></head><body><h1>Cobalt Core</h1></body></html>' > dist/index.html; \
    fi

# 2. Backend build stage (Go Monolith)
FROM golang:1.23-alpine AS go-builder
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .
COPY --from=web-builder /app/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /cobalt ./cmd/server

# 3. Final runtime stage
FROM alpine:3.20 AS runtime
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S cobalt && adduser -S cobalt -G cobalt

COPY --from=go-builder /cobalt /app/cobalt

USER cobalt

EXPOSE 8080 1883

ENTRYPOINT ["/app/cobalt"]
