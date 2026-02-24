# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/fraud-service/ services/fraud-service/

WORKDIR /build/services/fraud-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/fraudd ./cmd/fraudd

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder.
COPY --from=builder /bin/fraudd .

# Copy migrations.
COPY --from=builder /build/services/fraud-service/internal/infrastructure/postgres/migrations ./migrations

# Switch to non-root user.
USER appuser

# Expose gRPC and HTTP ports.
EXPOSE 8088 9088

ENTRYPOINT ["./fraudd"]
