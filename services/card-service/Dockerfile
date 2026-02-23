# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for better caching.
COPY go.mod go.sum* ./
COPY ../../pkg/ /app/pkg/

RUN go mod download

# Copy source code.
COPY . .

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /bin/card-service \
    ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user.
RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /bin/card-service .
COPY --from=builder /app/internal/infrastructure/persistence/migrations ./migrations

USER appuser

EXPOSE 8089 9089

ENTRYPOINT ["./card-service"]
