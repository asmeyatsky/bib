FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/payment-service/ services/payment-service/

WORKDIR /build/services/payment-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/paymentd ./cmd/paymentd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /bin/paymentd /usr/local/bin/paymentd
COPY --from=builder /build/services/payment-service/internal/infrastructure/persistence/postgres/migrations /migrations

EXPOSE 8086 9086

ENTRYPOINT ["paymentd"]
