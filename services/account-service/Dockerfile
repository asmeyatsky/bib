# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/account-service/ services/account-service/

WORKDIR /build/services/account-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/accountd ./cmd/accountd

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /bin/accountd /app/accountd

# Copy migrations
COPY --from=builder /build/services/account-service/internal/infrastructure/postgres/migrations /app/internal/infrastructure/postgres/migrations

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8082 9082

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9082/healthz || exit 1

ENTRYPOINT ["/app/accountd"]
