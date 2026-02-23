# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/card-service/ services/card-service/

WORKDIR /build/services/card-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/cardd ./cmd/cardd

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user.
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /bin/cardd .
COPY --from=builder /build/services/card-service/internal/infrastructure/postgres/migrations ./migrations

USER appuser

EXPOSE 8089 9089

ENTRYPOINT ["./cardd"]
