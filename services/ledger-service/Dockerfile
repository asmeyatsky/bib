FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/ledger-service/ services/ledger-service/

WORKDIR /build/services/ledger-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/ledgerd ./cmd/ledgerd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /bin/ledgerd /app/ledgerd
COPY --from=builder /build/services/ledger-service/internal/infrastructure/postgres/migrations /app/internal/infrastructure/postgres/migrations

EXPOSE 8081 9081

ENTRYPOINT ["ledgerd"]
