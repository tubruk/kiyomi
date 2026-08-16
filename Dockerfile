# Stage 1: Build the React frontend
FROM oven/bun:1.3.14-alpine AS frontend-builder
WORKDIR /app

# Copy root configurations and lockfile
COPY package.json bun.lock ./
# Copy web app files
COPY web/package.json web/bun.lock ./web/

# Install dependencies
RUN bun install --frozen-lockfile

# Copy web source code
COPY web/ ./web/

# Build frontend
RUN bun run build:ui

# Stage 2: Build the Go backend
FROM golang:1.26-alpine AS backend-builder
RUN apk add --no-cache bash
WORKDIR /app

# Copy Go dependency manifests
COPY go.mod go.sum go.work go.work.sum ./
COPY plugin-sdk/ ./plugin-sdk/

# Download Go dependencies
RUN go mod download

# Copy backend source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY scripts/ ./scripts/

# Copy built frontend assets from frontend-builder
COPY --from=frontend-builder /app/web/dist ./web/dist

# Run go generate to sync and embed frontend assets into the Go binary
RUN go generate ./pkg/webui/...

# Build-time arguments for versioning
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build Go binary with ldflags
RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/kiyomi \
    ./cmd/kiyomi

# Stage 3: Runner
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

# Copy binary from builder
COPY --from=backend-builder /app/kiyomi /usr/local/bin/kiyomi

# Expose default API port
EXPOSE 8080

# Environment defaults
ENV KIYOMI_HOME=/app/data
ENV KIYOMI_PORT=8080

# Run kiyomi
ENTRYPOINT ["/usr/local/bin/kiyomi"]
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD curl -f http://localhost:${KIYOMI_PORT}/api/v1/info || exit 1
