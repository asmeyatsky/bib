FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/fx-service/ services/fx-service/

WORKDIR /build/services/fx-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/fxd ./cmd/fxd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /bin/fxd /usr/local/bin/fxd
COPY --from=builder /build/services/fx-service/internal/infrastructure/postgres/migrations /migrations

USER app

EXPOSE 8083 9083

ENTRYPOINT ["fxd"]
