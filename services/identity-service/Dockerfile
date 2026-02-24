FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/identity-service/ services/identity-service/

WORKDIR /build/services/identity-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/identityd ./cmd/identityd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /bin/identityd /app/identityd
COPY --from=builder /build/services/identity-service/internal/infrastructure/postgres/migrations /app/internal/infrastructure/postgres/migrations

EXPOSE 8085 9085

ENTRYPOINT ["identityd"]
