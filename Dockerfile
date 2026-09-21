# Build stage
FROM golang:1.27.1-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS builder

WORKDIR /app

# Install git for version detection
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with version information
# Accept build args for version info, fall back to git describe if not provided
ARG VERSION
ARG COMMIT
ARG BUILD_DATE

RUN VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")} && \
    COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")} && \
    BUILD_DATE=${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")} && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w \
    -X github.com/sapcc/mosquitto-exporter/internal/version.Version=$VERSION \
    -X github.com/sapcc/mosquitto-exporter/internal/version.Commit=$COMMIT \
    -X github.com/sapcc/mosquitto-exporter/internal/version.BuildDate=$BUILD_DATE" \
    -o mosquitto-exporter .

# Final stage
FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

RUN apk --no-cache add ca-certificates

# Setup an unprivileged user
RUN addgroup -g 1000 appgroup && \
    adduser -D -u 1000 -G appgroup appuser

WORKDIR /app
RUN chown appuser:appgroup /app

USER appuser

# Copy the binary from builder stage
COPY --from=builder --chown=appuser:appgroup /app/mosquitto-exporter .

# Expose port
EXPOSE 9234

# Run the application
CMD ["./mosquitto-exporter"]
