FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/
COPY gateway/ gateway/

WORKDIR /build/gateway

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/gatewayd ./cmd/gatewayd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /bin/gatewayd /app/gatewayd

USER app

EXPOSE 8080

ENTRYPOINT ["/app/gatewayd"]
