#!/bin/bash
# Update all service Dockerfiles to use minimal alpine with wget for healthchecks

set -e

SERVICES=(
    "account-service"
    "fx-service"
    "deposit-service"
    "identity-service"
    "payment-service"
    "lending-service"
    "fraud-service"
    "card-service"
    "reporting-service"
)

for SERVICE in "${SERVICES[@]}"; do
    CMD_DIR=$(find "services/${SERVICE}/cmd" -mindepth 1 -maxdepth 1 -type d | head -1)
    CMD_NAME=$(basename "$CMD_DIR")
    
    # Default ports if not found
    case $SERVICE in
        "account-service") HTTP_PORT="8082"; GRPC_PORT="9082" ;;
        "fx-service") HTTP_PORT="8083"; GRPC_PORT="9083" ;;
        "deposit-service") HTTP_PORT="8084"; GRPC_PORT="9084" ;;
        "identity-service") HTTP_PORT="8085"; GRPC_PORT="9085" ;;
        "payment-service") HTTP_PORT="8086"; GRPC_PORT="9086" ;;
        "lending-service") HTTP_PORT="8087"; GRPC_PORT="9087" ;;
        "fraud-service") HTTP_PORT="8088"; GRPC_PORT="9088" ;;
        "card-service") HTTP_PORT="8089"; GRPC_PORT="9089" ;;
        "reporting-service") HTTP_PORT="8090"; GRPC_PORT="9090" ;;
    esac

    # Check if service has migrations
    MIGRATIONS=""
    if [ -d "services/${SERVICE}/internal/infrastructure/postgres/migrations" ]; then
        MIGRATIONS="COPY --from=builder /build/services/${SERVICE}/internal/infrastructure/postgres/migrations /app/internal/infrastructure/postgres/migrations"
    elif [ -d "services/${SERVICE}/migrations" ]; then
        MIGRATIONS="COPY --from=builder /build/services/${SERVICE}/migrations /app/internal/infrastructure/postgres/migrations"
    fi

    cat > "services/${SERVICE}/Dockerfile" << EOF
# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# Build Stage
# -----------------------------------------------------------------------------
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy shared packages first for better caching
COPY pkg/ pkg/

# Copy service
COPY services/${SERVICE}/ services/${SERVICE}/

WORKDIR /build/services/${SERVICE}

ENV GOWORK=off
RUN --mount=type=cache,target=/go/pkg/mod \\
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \\
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/${CMD_NAME} ./cmd/${CMD_NAME}

# -----------------------------------------------------------------------------
# Runtime Stage - Minimal Alpine
# -----------------------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /bin/${CMD_NAME} /app/${CMD_NAME}
${MIGRATIONS}

EXPOSE ${HTTP_PORT} ${GRPC_PORT}

ENTRYPOINT ["/app/${CMD_NAME}"]
EOF

    echo "Updated services/${SERVICE}/Dockerfile"
done

echo "All service Dockerfiles updated!"
