# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# Build Stage
# -----------------------------------------------------------------------------
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/lending-service/ services/lending-service/

WORKDIR /build/services/lending-service

ENV GOWORK=off
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/lendingd ./cmd/lendingd

# -----------------------------------------------------------------------------
# Runtime Stage - Minimal Alpine
# -----------------------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /bin/lendingd /app/lendingd
COPY --from=builder /build/services/lending-service/internal/infrastructure/postgres/migrations /app/internal/infrastructure/postgres/migrations

EXPOSE 8087 9087

ENTRYPOINT ["/app/lendingd"]
