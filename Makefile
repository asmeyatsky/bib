SERVICES := \
	services/ledger-service \
	services/account-service \
	services/fx-service \
	services/deposit-service \
	services/identity-service \
	services/payment-service \
	services/lending-service \
	services/fraud-service \
	services/card-service \
	services/reporting-service \
	gateway

PKGS := \
	pkg/money \
	pkg/events \
	pkg/postgres \
	pkg/kafka \
	pkg/auth \
	pkg/observability \
	pkg/iso20022 \
	pkg/testutil \
	pkg/tlsutil \
	pkg/residency \
	pkg/openbanking

ALL_MODULES := $(PKGS) $(SERVICES)

.PHONY: all lint test test-integration build proto docker-build docker-up docker-down test-e2e migrate-up migrate-down clean

all: lint test build

lint:
	@echo "==> Running linters..."
	@for mod in $(ALL_MODULES); do \
		echo "  Linting $$mod..."; \
		(cd $$mod && golangci-lint run ./...) || exit 1; \
	done

test:
	@echo "==> Running tests..."
	@for mod in $(ALL_MODULES); do \
		echo "  Testing $$mod..."; \
		(cd $$mod && go test -race -coverprofile=coverage.out ./...) || exit 1; \
	done

test-integration:
	@echo "==> Running integration tests..."
	@for svc in $(SERVICES); do \
		echo "  Integration testing $$svc..."; \
		(cd $$svc && go test -race -tags=integration ./...) || exit 1; \
	done

build:
	@echo "==> Building service binaries..."
	@mkdir -p bin
	@for svc in $(SERVICES); do \
		name=$$(basename $$svc); \
		echo "  Building $$name..."; \
		(cd $$svc && go build -o ../../bin/$$name ./cmd/$$(basename $$(find cmd -mindepth 1 -maxdepth 1 -type d | head -1))) || exit 1; \
	done

proto:
	@echo "==> Generating protobuf code..."
	buf generate

docker-build:
	@echo "==> Building Docker images..."
	@for svc in $(SERVICES); do \
		name=$$(basename $$svc); \
		echo "  Building Docker image for $$name..."; \
		docker build -t bib-$$name:latest -f $$svc/Dockerfile . || exit 1; \
	done

docker-up:
	@echo "==> Starting services..."
	docker compose up -d

docker-down:
	@echo "==> Stopping services..."
	docker compose down

test-e2e:
	@echo "==> Running end-to-end tests..."
	cd e2e && go test -race -tags=e2e -timeout=10m ./...

migrate-up:
	@echo "==> Running migrations up..."
	@for svc in $(SERVICES); do \
		name=$$(basename $$svc); \
		if [ -d "$$svc/migrations" ]; then \
			echo "  Migrating $$name up..."; \
			migrate -path $$svc/migrations -database "$${$$(echo $$name | tr '-' '_' | tr '[:lower:]' '[:upper:]')_DATABASE_URL}" up || exit 1; \
		fi; \
	done

migrate-down:
	@echo "==> Running migrations down..."
	@for svc in $(SERVICES); do \
		name=$$(basename $$svc); \
		if [ -d "$$svc/migrations" ]; then \
			echo "  Migrating $$name down..."; \
			migrate -path $$svc/migrations -database "$${$$(echo $$name | tr '-' '_' | tr '[:lower:]' '[:upper:]')_DATABASE_URL}" down 1 || exit 1; \
		fi; \
	done

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf bin/
	rm -rf gen/
	rm -rf tmp/
	@for mod in $(ALL_MODULES); do \
		rm -f $$mod/coverage.out; \
	done
	@echo "  Done."
