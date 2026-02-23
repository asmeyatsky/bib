FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files first for layer caching.
COPY go.mod go.sum* ./
COPY ../../pkg /app/pkg

ENV GOWORK=off
RUN go mod download

# Copy source code.
COPY . .

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /bin/reporting-service \
    ./cmd/server

# --- Runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /bin/reporting-service /bin/reporting-service
COPY --from=builder /app/internal/infrastructure/persistence/migrations /migrations

USER appuser

EXPOSE 8090 9090

ENTRYPOINT ["/bin/reporting-service"]
