# ==========================================
# Stage 1: Build Stage
# ==========================================
FROM golang:1.27-alpine AS builder

# Install build dependencies and CA certificates
RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /app

# Accept build argument for server path (defaults to ./cmd/server)
ARG CMD_SERVER_PATH=./cmd/server

# Cache Go modules layer separately from source code
COPY go.mod go.sum ./
RUN go mod download

# Copy application source
COPY . .

# Build static binary targeting entrypoint path
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/server ${CMD_SERVER_PATH}

# ==========================================
# Stage 2: Minimal Production Runtime
# ==========================================
FROM alpine:3.20 AS final

# Create a non-privileged user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# 1. Accept PORT as a build argument with a default fallback
ARG PORT=8080

# 2. Export it as an ENV variable so the Go binary can read os.Getenv("PORT") at runtime
ENV PORT=${PORT}

# Copy CA certificates and timezone data from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/server /app/server

# Set ownership to non-root user
RUN chown -R appuser:appgroup /app

# Enforce security context: switch to non-root user
USER appuser

# Expose target service port
EXPOSE ${PORT}

# Run binary
ENTRYPOINT ["/app/server"]