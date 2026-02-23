# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better caching.
COPY go.mod go.sum ./
COPY ../../pkg/ /pkg/

ENV GOWORK=off
RUN go mod download

# Copy source code.
COPY . .

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /bin/fraud-service \
    ./cmd/server

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder.
COPY --from=builder /bin/fraud-service .

# Copy migrations.
COPY --from=builder /app/internal/infrastructure/persistence/migration ./migrations

# Switch to non-root user.
USER appuser

# Expose gRPC and HTTP ports.
EXPOSE 8088 9088

ENTRYPOINT ["./fraud-service"]
