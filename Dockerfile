# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with static linking
RUN CGO_ENABLED=1 GOOS=linux go build -a \
    -ldflags '-linkmode external -extldflags "-static" -s -w' \
    -tags 'sqlite_omit_load_extension' \
    -o fustgo \
    ./cmd/fustgo

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/fustgo /app/fustgo

# Copy default configuration
COPY configs/default.yaml /app/config.yaml

# Create necessary directories
RUN mkdir -p /data /var/log/fustgo /opt/fustgo/plugins

# Create non-root user
RUN addgroup -g 1000 fustgo && \
    adduser -D -s /bin/sh -u 1000 -G fustgo fustgo && \
    chown -R fustgo:fustgo /app /data /var/log/fustgo

USER fustgo

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/fustgo"]
CMD ["--config", "/app/config.yaml"]
