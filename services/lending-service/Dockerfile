FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/lending-service/ services/lending-service/

WORKDIR /build/services/lending-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/lendingd ./cmd/lendingd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /bin/lendingd /app/lendingd
COPY --from=builder /build/services/lending-service/migrations /app/internal/infrastructure/postgres/migrations

USER app

EXPOSE 8087 9087

ENTRYPOINT ["/app/lendingd"]
