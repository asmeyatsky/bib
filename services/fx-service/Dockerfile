# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy shared packages first for better caching.
COPY pkg/ pkg/

# Copy go.mod/go.sum for dependency download.
COPY services/fx-service/go.mod services/fx-service/go.sum* services/fx-service/

WORKDIR /src/services/fx-service
ENV GOWORK=off
RUN go mod download

# Copy source.
COPY services/fx-service/ .

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /bin/fxd ./cmd/fxd

# ---- Runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S app -G app
USER app

COPY --from=builder /bin/fxd /usr/local/bin/fxd
COPY --from=builder /src/services/fx-service/internal/infrastructure/postgres/migrations /migrations

EXPOSE 8083 9083

ENTRYPOINT ["fxd"]
