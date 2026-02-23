FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/deposit-service/ services/deposit-service/

WORKDIR /build/services/deposit-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/depositd ./cmd/depositd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

COPY --from=builder /bin/depositd /usr/local/bin/depositd
COPY --from=builder /build/services/deposit-service/internal/infrastructure/postgres/migrations /migrations

EXPOSE 8084 9084

ENTRYPOINT ["depositd"]
