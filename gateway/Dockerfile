FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy go workspace and shared packages first for layer caching.
COPY go.work go.work
COPY pkg/ pkg/
COPY gateway/ gateway/

WORKDIR /src/gateway

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/gatewayd ./cmd/gatewayd

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S bib && adduser -S bib -G bib

COPY --from=builder /bin/gatewayd /usr/local/bin/gatewayd

USER bib

EXPOSE 8080

ENTRYPOINT ["gatewayd"]
