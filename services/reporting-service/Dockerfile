# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/reporting-service/ services/reporting-service/

WORKDIR /build/services/reporting-service

ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/reportingd ./cmd/reportingd

# --- Runtime stage ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /bin/reportingd /bin/reportingd
COPY --from=builder /build/services/reporting-service/internal/infrastructure/postgres/migrations /migrations

USER appuser

EXPOSE 8090 9090

ENTRYPOINT ["/bin/reportingd"]
