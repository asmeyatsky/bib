# ---------------------------------------------------------------------------
# Build stage
# ---------------------------------------------------------------------------
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy shared packages first for better caching.
COPY pkg/ /src/pkg/

# Copy lending service module.
COPY services/lending-service/go.mod services/lending-service/go.sum* /src/services/lending-service/
WORKDIR /src/services/lending-service
ENV GOWORK=off
RUN go mod download

COPY services/lending-service/ /src/services/lending-service/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /lending-service ./cmd/server

# ---------------------------------------------------------------------------
# Runtime stage
# ---------------------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

COPY --from=builder /lending-service /usr/local/bin/lending-service
COPY services/lending-service/migrations/ /migrations/

USER app

EXPOSE 8087 9087

ENTRYPOINT ["lending-service"]
