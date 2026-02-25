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
COPY services/fraud-service/ services/fraud-service/

WORKDIR /build/services/fraud-service

ENV GOWORK=off
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/fraudd ./cmd/fraudd

# -----------------------------------------------------------------------------
# Runtime Stage - Minimal Alpine
# -----------------------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /bin/fraudd /app/fraudd
COPY --from=builder /build/services/fraud-service/internal/infrastructure/postgres/migrations /app/internal/infrastructure/postgres/migrations

EXPOSE 8088 9088

ENTRYPOINT ["/app/fraudd"]
