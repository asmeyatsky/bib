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
COPY services/card-service/ services/card-service/

WORKDIR /build/services/card-service

ENV GOWORK=off
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/cardd ./cmd/cardd

# -----------------------------------------------------------------------------
# Runtime Stage - Minimal Alpine
# -----------------------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /bin/cardd /app/cardd
COPY --from=builder /build/services/card-service/internal/infrastructure/postgres/migrations /app/internal/infrastructure/postgres/migrations

EXPOSE 8089 9089

ENTRYPOINT ["/app/cardd"]
