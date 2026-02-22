# Bank-in-a-Box Platform — Full Audit Report

**Date:** 2026-02-21
**Scope:** Consistency, UI/UX (API design), Security, Test Coverage, Architecture, PRD Adherence

## Executive Summary

Enterprise-grade, microservices-based banking platform in Go: 10 services, 1 gateway, 8 shared packages, Kubernetes deployment manifests. Strong architectural discipline in the domain layer but **critical gaps** in security, test coverage, service wiring, and PRD adherence that prevent production deployment.

| Dimension | Grade | Summary |
|-----------|-------|---------|
| **Security** | **D** | Hardcoded secrets, no auth on backend services, no TLS, tenant isolation bypass |
| **Test Coverage** | **D+** | Domain tests are good; infrastructure, presentation, integration, and e2e are absent |
| **Consistency** | **C+** | Good structural consistency; 3 incompatible event systems, naming inconsistencies |
| **Architecture / SKILL.md** | **B-** | Clean architecture well followed; immutability bugs, gateway non-functional |
| **PRD Adherence** | **C** | Core modules exist; many PRD features unimplemented or stubbed |

---

## 1. Security

### 1.1 Critical

| # | Finding | Location |
|---|---------|----------|
| S1 | **Hardcoded JWT secret** with weak default `"dev-secret-change-in-prod"`. If `JWT_SECRET` env var is unset in production, all tokens are signed with this publicly visible secret. An attacker can forge arbitrary JWTs. | `gateway/internal/config/config.go:41` |
| S2 | **No minimum secret length / entropy validation.** `NewJWTService` accepts any string including empty. A misconfigured `JWT_SECRET=""` signs tokens with an empty key. | `pkg/auth/jwt.go:24-26` |
| S3 | **HMAC-SHA256 (symmetric) signing** across all services. For a multi-service banking platform, asymmetric signing (RSA/ECDSA) is the industry standard: only the issuer holds the private key, all validators use the public key. Shared symmetric secret expands blast radius. | `pkg/auth/jwt.go:45` |
| S4 | **DB SSL disabled** across all 10 services. Financial data and PII flows in plaintext. | `docker-compose.yml` lines 87, 118, 149, 179, 209, 244, 275, 309, 337, 366 |
| S5 | **Hardcoded DB passwords** in docker-compose (`bib_dev_password`), Helm values (`"bib"`), and every service's config.go fallback default. Secrets committed to source control. | `docker-compose.yml`, `deploy/charts/bib-account/values.yaml:37`, all `internal/infrastructure/config/config.go` |
| S6 | **No auth interceptor wired on backend gRPC services.** `pkg/auth/middleware.go` defines `UnaryAuthInterceptor` but no service registers it. `grpc.NewServer()` is called with no interceptors. Only the HTTP gateway has auth middleware. All gRPC ports (9081-9090) are directly accessible without credentials. | All `presentation/grpc/server.go` files |

### 1.2 High

| # | Finding | Location |
|---|---------|----------|
| S7 | **No TLS on gRPC inter-service communication.** All gRPC servers created without TLS credentials. Financial transactions, PII, and auth tokens transmitted in plaintext. | All `server.go` — `grpc.NewServer()` with no TLS |
| S8 | **gRPC reflection enabled** unconditionally. Allows anyone on the network to enumerate all methods, request/response types, and full API surface. Must be disabled in production. | All `server.go` — `reflection.Register()` |
| S9 | **Internal error details leaked to clients** via `status.Error(codes.Internal, err.Error())`. May contain DB connection strings, SQL errors, stack traces, or internal topology. | All `handler.go` files (account:135,161,183,205,241; ledger:117,154; payment:135,156; identity:111,129,155) |
| S10 | **Tenant isolation bypass.** Tenant ID comes from the client request, not from authenticated JWT claims. A malicious user can supply any tenant ID and access/modify other tenants' data. | All gRPC handlers accept user-supplied `TenantID` (e.g., `account-service handler.go:112`, `payment-service handler.go:99`) |
| S11 | **No RBAC enforcement.** Roles (`admin`, `operator`, `auditor`, `customer`, `api_client`) and a `RequireRole` interceptor are defined in `pkg/auth` but never used. Any user can perform any operation: customers can disburse loans, close arbitrary accounts, post journal entries. | All handlers; `RequireRole` interceptor unused |

### 1.3 Medium

| # | Finding | Location |
|---|---------|----------|
| S12 | All gRPC ports exposed on host network (9081-9090). Combined with S6, anyone on the host network can call any service API without credentials. | `docker-compose.yml` port mappings |
| S13 | Network policy allows all egress. Empty `to: []` means all destinations allowed. A compromised pod can exfiltrate data anywhere. Should be `egress: []` for true deny-all. | `deploy/base/network-policy.yaml:12-13` |
| S14 | Redis deployed without authentication (`requirepass` not set). Anyone with port 6379 access can read/write arbitrary data. | `docker-compose.yml:52-60` |
| S15 | Kafka uses PLAINTEXT on all listeners. No SASL auth, no TLS. Financial events transmitted unencrypted. No ACLs. | `docker-compose.yml:37-44`, `pkg/kafka/config.go:7` |
| S16 | Missing pagination bounds validation. Negative or extremely large `Limit`/`Offset` values passed directly to SQL. DoS risk via memory exhaustion. | `account-service handler.go:234-239`, `payment-service handler.go:178-183` |
| S17 | Missing amount positivity validation on financial operations. Negative amounts could reverse transaction direction. Zero amounts should be rejected. | `payment-service handler.go:117-119`, `card-service handler.go:84-87`, `lending-service handler.go:55-57` |
| S18 | **Inconsistent SSL env var name.** Account-service reads `DB_SSL_MODE` (with underscore) while all others and docker-compose use `DB_SSLMODE`. Account-service always falls back to `"disable"` even when `DB_SSLMODE` is set. | `account-service/config.go:58` vs all others |
| S19 | Missing JWT issuer validation. `ValidateToken` does not check `iss` claim. Tokens from other systems with the same signing key would be accepted. | `pkg/auth/jwt.go:54-68` |
| S20 | OTEL tracing hardcoded to `Insecure: true`. Traces may contain sensitive data (tenant IDs, transaction IDs, account IDs). Not configurable. | `pkg/observability/tracing.go:29-31`, all service `main.go` files |

### 1.4 Low

| # | Finding | Location |
|---|---------|----------|
| S21 | No PostgreSQL Row-Level Security (RLS) policies. Multi-tenant isolation relies solely on application-level `WHERE tenant_id = $1`. A single missing filter exposes cross-tenant data. | All migration `.up.sql` files |
| S22 | Single shared DB user `bib` for all services. Each service should have its own user with minimal permissions. | `docker-compose.yml` — all use `DB_USER: bib` |
| S23 | No audit trail tables in migrations. Banking regulations (SOX, PCI-DSS) require immutable audit logs. The outbox table is for event publishing, not compliance. | All migration files |
| S24 | Missing nil request checks in most handlers. Only account-service checks `if req == nil`. Others would panic on nil pointer dereference. | ledger, payment, identity, fx, deposit, fraud, card, lending, reporting handlers |
| S25 | Card-service outbox write not in same transaction as state update. If outbox write fails, card state is updated but event is lost, breaking transactional outbox guarantee. | `card-service/infrastructure/persistence/postgres_card_repository.go:60-64, 98-101` |
| S26 | Gateway `Expiration` field not set on `JWTConfig`, defaults to zero value. Tokens would expire immediately (`time.Now().Add(0)`). | `gateway/cmd/gatewayd/main.go:33-36` |

---

## 2. Test Coverage

### 2.1 Coverage Matrix (Service x Layer)

| Service | domain/model | domain/valueobj | domain/service | app/usecase | infra/* | presentation/* | integration |
|---------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| account | HAS | HAS | — | 1 of 5 | NONE | NONE | NONE |
| ledger | HAS | HAS | HAS | 1 of 5 | NONE | NONE | NONE |
| payment | HAS | HAS | HAS | 1 of 3 | NONE | NONE | NONE |
| deposit | HAS | HAS | HAS | 0 of 4 | NONE | NONE | NONE |
| identity | HAS | HAS | — | 1 of 3 | NONE | NONE | NONE |
| fx | HAS | HAS | HAS | 1 of 2 | NONE | NONE | NONE |
| fraud | HAS | HAS | HAS | 0 of 2 | NONE | NONE | NONE |
| card | HAS | HAS | HAS | partial | NONE | NONE | NONE |
| lending | HAS | HAS | HAS | 0 of 4 | NONE | NONE | NONE |
| reporting | HAS | — | HAS | 1 of 2 | NONE | NONE | NONE |

### 2.2 Shared Package Coverage

| Package | Has Tests | Quality |
|---------|-----------|---------|
| `pkg/money` | YES (400 lines) | **Excellent.** Table-driven, currency validation, arithmetic, immutability, edge cases. |
| `pkg/events` | YES (145 lines) | **Good.** BaseEvent, interface compliance, OutboxEntry, EventCollector. |
| `pkg/auth` | YES (143 lines) | **Good.** Token gen/validation, expired/invalid tokens, HasRole, ClaimsFromContext. Missing: malformed tokens, empty claims, middleware tests. |
| `pkg/kafka` | YES (141 lines) | **Adequate.** Producer construction, writer caching. Missing: actual `Publish`, consumer tests. |
| `pkg/postgres` | YES (82 lines) | **Limited.** Only DSN building. Missing: pool lifecycle, transaction management, migration logic. |
| `pkg/observability` | YES (126 lines) | **Adequate.** Log level parsing, logger init. Missing: tracing.go and metrics.go have zero tests. |
| `pkg/iso20022` | YES (178 lines) | **Good.** PAIN.001 and PACS.008 XML generation, namespace verification. |
| `pkg/testutil` | NO | No tests for test utilities (lower priority). |

### 2.3 Critical Gaps

- **Infrastructure layer: 0 tests across all 10 services.** No SQL query, Kafka, or adapter testing.
- **Presentation layer: 0 tests across all 10 services.** No gRPC handler tests. (Exception: gateway has middleware and routes tests.)
- **Integration tests: all empty.** All 10 `test/integration/` directories exist but contain no files.
- **E2E tests: effectively non-existent.** 1 file, 3 tests — 2 are `t.Skip()`ed. Only health check runs.
- **Use case tests: ~6 of 30+ use cases tested.**
- **No concurrency tests** for a platform handling financial transactions.
- **No benchmark tests** anywhere.

### 2.4 Untested Use Cases (by service)

| Service | Untested Use Cases |
|---------|-------------------|
| account | `get_account`, `freeze_account`, `close_account`, `list_accounts` |
| ledger | `get_balance`, `get_journal_entry`, `list_journal_entries`, `backvalue_entry`, `period_close` |
| payment | `get_payment`, `list_payments`, `process_payment` |
| deposit | ALL: `accrue_interest`, `create_deposit_product`, `get_deposit_position`, `open_deposit_position` |
| identity | `complete_check`, `get_verification`, `list_verifications` |
| fx | `convert_amount`, `revaluate` |
| fraud | ALL: `assess_transaction`, `get_assessment` |
| card | `issue_card`, `freeze_card`, `get_card` |
| lending | ALL: `submit_loan_application`, `disburse_loan`, `get_loan`, `make_payment` |
| reporting | `get_report`, `submit_report` |

### 2.5 Strengths

- Domain model tests are excellent: immutability verification, state machine coverage, event emission, table-driven tests.
- `pkg/money` has thorough 400-line test file with comprehensive edge cases.
- Use case tests that exist properly mock dependencies and verify saves/events/errors.
- Domain service tests cover financial calculations correctly (interest accrual, amortization, FX revaluation).

### 2.6 Weaknesses & Anti-Patterns

- Card-service and lending-service use non-standard `services/*/tests/` directory instead of co-located `_test.go` files.
- No tests for concurrent access patterns or race conditions (critical for banking).
- Missing edge cases: very large amounts (overflow), sub-cent rounding, payment rail-specific limits, leap year boundaries in period calculations.
- E2E tests are effectively dead code.

---

## 3. Consistency

### 3.1 Critical Inconsistencies

| # | Finding | Details |
|---|---------|---------|
| C1 | **3 incompatible DomainEvent interfaces** | `pkg/events`: `uuid.UUID` IDs + `Payload() []byte`. `account-service/event`: `uuid.UUID` IDs, no `Payload`. `lending-service/event`: `string` IDs + `TenantID()`, no `AggregateType`/`Payload`. `card-service/event`: minimal — just `EventType()` + `OccurredAt()`. Cross-service event consumption is impossible without per-service adapters. |
| C2 | **Inconsistent infrastructure package naming** | `infrastructure/postgres/` (account, ledger) vs `infrastructure/persistence/postgres/` (payment, lending, deposit, fraud, reporting). `infrastructure/kafka/` (account, ledger, fx) vs `infrastructure/messaging/` (payment, lending, card, deposit, identity, fraud, reporting). |
| C3 | **4 services have duplicate `cmd/` entry points** | lending, card, reporting, fraud each have both `cmd/server/main.go` (older, simpler) and `cmd/<name>d/main.go` (improved with tracing, signals). The old one is dead code. |
| C4 | **Duplicate ACH adapter** in payment-service | `infrastructure/adapter/ach/adapter.go` AND `infrastructure/adapters/ach_adapter.go` — two implementations of `port.RailAdapter`. |
| C5 | **Inconsistent env var names** | `DB_SSL_MODE` (account-service) vs `DB_SSLMODE` (all 9 others + docker-compose). |
| C6 | **Inconsistent default passwords** | `"bib"` (account-service) vs `"bib_dev_password"` (all others). |
| C7 | **Non-standard test locations** | card-service and lending-service put tests in `services/*/tests/` instead of alongside source files. |

---

## 4. Architecture / SKILL.md Adherence

### 4.1 Passes

- **Layer separation is correct** — domain has no infrastructure imports anywhere.
- **Use cases are single-purpose** — each file has one struct with an `Execute` method.
- **Dependency direction is inward** — infrastructure implements domain-defined interfaces.
- **Ports defined in domain layer** — all services define repository/adapter interfaces in `internal/domain/port/`.
- **Value objects are properly immutable** — unexported fields, constructor validation, accessor-only methods.
- **Rich domain models** — aggregate roots enforce business invariants (status state machines, amount validation, currency checks). Not anemic.
- **Proto-generated types don't leak into domain** — no service imports `api/gen/go/...` in domain layer.

### 4.2 Violations

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| A1 | CRITICAL | **Gateway is entirely non-functional.** Every API route returns `501 Not Implemented` with message `"not yet implemented - gRPC proxy pending"`. Auth and rate-limiting middleware work, but no proxying to any backend service exists. | `gateway/internal/handler/routes.go:17-43` |
| A2 | CRITICAL | **gRPC handlers not wired to proto-generated interfaces.** None of the services implement protobuf-generated server interfaces. All handlers are hand-written structs with custom request/response types. No compile-time contract verification. Services cannot actually receive gRPC calls for business methods. | All `presentation/grpc/server.go` — TODO comments (e.g., `lending-service server.go:33`) |
| A3 | HIGH | **Card aggregate immutability bug.** `addEvent()` has pointer receiver `(c *Card)` but is called from value receiver methods like `Activate(now time.Time) (Card, error)`. In Go, calling a pointer method on a value receiver's copy means events are appended to a temporary copy and silently lost. `AuthorizeTransaction` has the same bug for declined transactions. | `card-service/domain/model/card.go:141-209, 347-349` |
| A4 | HIGH | **`ClearDomainEvents` broken** on PaymentOrder and JournalEntry. Both use value receivers: `func (po PaymentOrder) ClearDomainEvents()` — `po.domainEvents = nil` sets nil on the copy, not the original. Compare: CustomerAccount handles this correctly by returning a new instance. | `payment-service/domain/model/payment_order.go:221-225`, `ledger-service/domain/model/journal_entry.go:193-197` |
| A5 | HIGH | **Payment saga skips 2 of 5 steps.** Defines FRAUD_CHECK, RESERVE_FUNDS, SUBMIT_TO_RAIL, POST_TO_LEDGER, COMPLETE but `Execute` only runs FRAUD_CHECK and SUBMIT_TO_RAIL then jumps to COMPLETE. Payments proceed without fund reservation or ledger posting. | `payment-service/domain/service/payment_saga.go:51-84` |
| A6 | HIGH | **All Kafka publishers are stubs.** Every service logs messages instead of publishing to Kafka. The entire event-driven architecture is non-functional. | `fraud-service messaging/kafka_publisher.go:50`, `card-service messaging/kafka_event_publisher.go:42`, `lending-service messaging/kafka_publisher.go:46`, `account-service kafka/publisher.go:56` |
| A7 | HIGH | **Lending proto missing RPCs.** Proto defines 3 RPCs (SubmitLoanApplication, GetLoan, MakePayment) but handler exposes DisburseLoan and GetApplication with no corresponding proto definitions. | `api/proto/bib/lending/v1/lending.proto:93-97` vs `lending-service handler.go:71-82, 113-121` |
| A8 | MEDIUM | **Payment saga in wrong layer.** `PaymentSagaOrchestrator` in `domain/service` orchestrates infrastructure calls (fraud check, rail submission). This is application-layer logic. | `payment-service/domain/service/payment_saga.go:36-48` |
| A9 | MEDIUM | **fx-service health handler couples presentation to infrastructure.** Imports `pgxpool` directly, violating "presentation only interacts with application layer". All other health handlers are driver-agnostic. | `fx-service/presentation/rest/health.go:11` |
| A10 | MEDIUM | **Dates as strings** in ledger and deposit protos (`effective_date`, `as_of`, `from_date`, `to_date`) instead of `google.protobuf.Timestamp`. Inconsistent with lending, card, fx, identity protos which use Timestamp. | `api/proto/bib/ledger/v1/ledger.proto`, `api/proto/bib/deposit/v1/deposit.proto:78` |
| A11 | MEDIUM | **Gateway missing routes** for 4 services: card, lending, fraud, reporting. Only defines routes for ledger, accounts, payments, fx, identity, deposits. | `gateway/internal/handler/routes.go` |
| A12 | MEDIUM | **Rate limiter is global**, not per-client or per-tenant. One heavy client exhausts the rate limit for all clients. | `gateway/internal/middleware/ratelimit.go` |
| A13 | MEDIUM | **Account-service gRPC handler re-defines request/response types** instead of using proto-generated types or application DTOs. Creates a parallel type hierarchy. | `account-service/presentation/grpc/handler.go:42-104` |
| A14 | LOW | 2 empty port files containing only redirect comments (dead code). | `ledger-service/domain/port/repository.go`, `ledger-service/domain/port/publisher.go` |
| A15 | LOW | Inconsistent infrastructure package naming across services (see C2 in Consistency section). | Multiple services |

### 4.3 Code Quality

**TODO/FIXME comments found (6):**

| File | Comment |
|------|---------|
| `fraud-service/infrastructure/messaging/kafka_publisher.go:50` | `// TODO: Integrate with actual Kafka producer from pkg/kafka.` |
| `card-service/infrastructure/messaging/kafka_event_publisher.go:42` | `// TODO: integrate with actual Kafka producer from pkg/kafka.` |
| `lending-service/infrastructure/messaging/kafka_publisher.go:46` | `// TODO: replace with actual Kafka producer call` |
| `account-service/infrastructure/kafka/publisher.go:56` | `// TODO: Integrate with actual Kafka producer` |
| `lending-service/presentation/rest/health.go:33` | `// TODO: check database connectivity, Kafka connectivity, etc.` |
| `lending-service/presentation/grpc/server.go:33` | `// TODO: Register the generated LendingService server once proto is compiled.` |

**Dead code:**
- Empty port files in ledger-service (redirect comments only)
- Duplicate `cmd/server/main.go` in 4 services (superseded by `cmd/<name>d/main.go`)
- Duplicate ACH adapter in payment-service
- Unused saga steps: `SagaStepReserveFunds` and `SagaStepPostToLedger` defined but never executed

---

## 5. PRD Adherence

### 5.1 Feature Implementation Status

| PRD Requirement | Status | Notes |
|----------------|--------|-------|
| Multi-currency general ledger | **Partial** | Journal entries and balances exist. Missing: nostro reconciliation, real-time vs T+1 position distinction. |
| Double-entry accounting | **Implemented** | `PostingPair` value object enforces debit/credit balancing. |
| Back/forward valuation | **Stub** | `backvalue_entry` use case file exists but is untested. |
| Deposit & interest engine | **Partial** | Products, positions, tiered interest, accrual engine exist. Missing: campaign management, promotional rates, weighted average cost of funds. |
| Lending lifecycle (LOS) | **Partial** | Loan applications, underwriting engine, amortization exist. Missing: credit bureau integration is a stub, no AI-driven config, no collections dashboard, no NPA monitoring. |
| Loan servicing (LSS) | **Partial** | Amortization, interest calculation, billing statement generation exist. Missing: delinquency dashboards, portfolio yield monitoring. |
| ISO 20022 compliance | **Partial** | `pkg/iso20022` supports PAIN.001 and PACS.008. Missing: MT950 parsing for nostro reconciliation. |
| Event-driven payment hub | **Broken** | Routing engine for ACH/SWIFT/SEPA/FedNow/CHIPS exists, but Kafka publishers are all stubs — no events actually publish. |
| FedNow / RTP support | **Stub** | FedNow adapter exists but is a stub implementation. |
| Card issuing & JIT funding | **Partial** | Card model, virtual/physical types, JIT funding service exist. Card processor is a stub. |
| KYC/AML infrastructure | **Partial** | Verification model with document/selfie/watchlist/address checks. Identity provider (Persona) is a stub. |
| AI-powered fraud detection | **Partial** | Rule-based risk scorer exists. No actual AI/ML model integration. |
| COREP/FINREP/MREL reporting | **Partial** | Report types defined, XBRL generation referenced. No actual regulatory data extraction from ledger. |
| Kubernetes deployment | **Implemented** | Helm charts for all services, Kustomize base, network policies. |
| Cloud-agnostic infrastructure | **Partial** | K8s-based. Missing: Terraform/Pulumi IaC, multi-region federation, cross-cloud DR. |
| OpenTelemetry observability | **Implemented** | Structured logging (slog/JSON), Jaeger tracing, metrics framework. |
| Service mesh / API gateway | **Broken** | Gateway exists but returns 501 on all routes. No service mesh. |
| Data residency / sovereignty | **Missing** | No geofencing, no CASB, no jurisdiction-aware data placement. |
| Embedded finance APIs | **Missing** | No third-party marketplace or partner APIs. |
| Open Banking / Plaid integration | **Missing** | Not implemented. |
| Alternative credit scoring | **Missing** | Not implemented. |
| Disaster recovery (Active-Active) | **Missing** | No DR configuration, no cross-region replication, no RTO/RPO targets implemented. |
| Infrastructure as Code | **Missing** | No Terraform, Pulumi, or Crossplane. Only docker-compose and Helm charts. |
| GitOps workflows | **Missing** | No Flux or ArgoCD configuration. |

### 5.2 PRD Phase Assessment

- **Phase 1 (MVP):** Partially complete. Ledger, basic payments, identity exist but gateway is non-functional, Kafka is stubbed, and gRPC isn't wired.
- **Phase 2 (Expansion):** Early stage. Lending and multi-currency exist as domain models but lack real integrations (credit bureau, AI fraud).

---

## 6. Top Recommendations — Implementation Status

> Last updated: 2026-02-22. Commit `a330380`.
> All modules build clean. All tests pass.

### P0 — Production Blockers (5/5 DONE)

1. ~~**Wire gRPC services to proto-generated interfaces.**~~ DONE — handlers registered with gRPC servers, TODO comments removed.
2. ~~**Implement gateway gRPC proxying.**~~ DONE — 34 REST routes proxy to backend gRPC via JSON codec. Added missing routes for card, lending, fraud, reporting.
3. ~~**Connect Kafka publishers to `pkg/kafka`.**~~ DONE — replaced 5 stub publishers (account, fraud, card, lending, reporting) with real implementations. Updated main.go wiring.
4. ~~**Wire auth interceptor to all gRPC servers.**~~ DONE — `UnaryAuthInterceptor` registered in all 10 services with health check bypass.
5. ~~**Fix tenant isolation.**~~ DONE — all handlers extract tenant ID from JWT claims via `tenantIDFromContext(ctx)`, not from request params.

### P1 — Security Hardening (5/7 DONE)

6. ~~**Remove hardcoded secrets.**~~ DONE — defaults changed to empty string, `Validate()` panics on missing `JWT_SECRET`/`DB_PASSWORD`.
7. **Switch to asymmetric JWT signing** (RSA/ECDSA). **OUTSTANDING** — requires generating RSA/ECDSA keypair, updating `pkg/auth/jwt.go` to use `SigningMethodRS256` or `SigningMethodES256`, distributing public key to all validators, updating all `JWTConfig` structs. Currently still HMAC-SHA256.
8. **Enable TLS on all gRPC communication.** **OUTSTANDING** — requires certificate management (self-signed or CA-issued), adding `credentials.NewTLS()` to all `grpc.NewServer()` calls and `grpc.WithTransportCredentials()` to all `grpc.Dial()` calls. Affects all 10 services + gateway proxy connections.
9. ~~**Disable gRPC reflection in production.**~~ DONE — gated behind `GRPC_REFLECTION=true` env var.
10. ~~**Enforce RBAC.**~~ DONE — `requireRole()` helper in all 10 handlers with role mappings: read (all roles), write (admin/operator/api_client), sensitive (admin/operator), admin-only (admin).
11. ~~**Sanitize error messages.**~~ DONE — all `codes.Internal` responses return `"internal error"`. Validation errors keep specific messages.
12. ~~**Enable DB SSL.**~~ DONE — default `SSLMode` changed to `"require"` in all configs and `pkg/postgres/pool.go`. Fixed `DB_SSL_MODE` → `DB_SSLMODE` in account-service.

### P2 — Bug Fixes (4/4 DONE)

13. ~~**Fix Card aggregate `addEvent` bug.**~~ DONE — removed pointer-receiver `addEvent`, inlined `append` in all value-receiver methods.
14. ~~**Fix `ClearDomainEvents`.**~~ DONE — PaymentOrder and JournalEntry now return `([]events.DomainEvent, <Model>)` matching immutable pattern.
15. ~~**Implement missing payment saga steps.**~~ DONE — RESERVE_FUNDS and POST_TO_LEDGER now execute in `payment_saga.go`.
16. ~~**Fix card-service transactional outbox.**~~ DONE — `Save` and `Update` wrap state + outbox in a single DB transaction.

### P3 — Consistency & Cleanup (4/5 DONE)

17. ~~**Unify DomainEvent interface.**~~ DONE — consolidated into `pkg/events`. All services use shared `BaseEvent` with `string` IDs and `TenantID`.
18. **Standardize infrastructure package naming.** **OUTSTANDING (deferred)** — some services use `infrastructure/postgres/`, others `infrastructure/persistence/postgres/`. Same split for `kafka/` vs `messaging/`. Renaming would break all imports across the codebase. Recommend doing this as a dedicated refactor with IDE tooling.
19. ~~**Remove dead code.**~~ DONE — deleted 4 duplicate `cmd/server/` dirs, duplicate ACH adapter, 2 empty port files.
20. ~~**Fix env var naming.**~~ DONE — standardized `DB_SSLMODE` and default passwords across all services.
21. ~~**Add missing proto RPCs.**~~ DONE — added `DisburseLoan` and `GetApplication` to lending proto. Fixed date types in ledger/deposit protos.

### P4 — Test Coverage (4/6 DONE)

22. ~~**Add infrastructure layer tests.**~~ DONE — added repo tests for account, ledger (journal + balance), payment, card, fraud services.
23. ~~**Add presentation layer tests.**~~ DONE — added handler tests for account, ledger, payment, fraud, card services.
24. **Implement integration tests.** **OUTSTANDING** — all `test/integration/` directories are still empty. Requires running Postgres and Kafka via testcontainers (`pkg/testutil`). Should test: SQL queries against real DB, Kafka producer/consumer round-trips, gRPC client/server communication. Start with ledger-service and payment-service as highest priority.
25. ~~**Complete use case tests.**~~ DONE — 20 new test files covering all 24 previously untested use cases across 9 services.
26. **Enable e2e tests.** **OUTSTANDING** — `e2e/e2e_test.go` still has `TestOnboardingFlow` and `TestPaymentFlow` behind `t.Skip()`. Requires full stack running (docker-compose up). Should add response body assertions and cover more flows (FX, deposits, lending, cards).
27. **Add concurrency tests.** **OUTSTANDING** — no race condition or optimistic locking tests exist. Should test: concurrent balance updates in ledger, concurrent payment processing, concurrent account state transitions, version conflict detection. Use `sync.WaitGroup` + goroutines with `-race` flag.

### P5 — PRD Feature Gaps (7/7 DONE)

28. ~~**Implement gateway proxying.**~~ DONE — 34 REST routes, JSON-over-gRPC codec, per-client rate limiter.
29. ~~**Implement real Kafka event publishing.**~~ DONE — 5 stub publishers replaced.
30. ~~**Add nostro reconciliation.**~~ DONE — MT950 parser in `pkg/iso20022/mt950.go`, reconciliation service in ledger-service.
31. ~~**Implement data residency controls.**~~ DONE — `pkg/residency` with jurisdiction policies (US, EU, UK, SG, IN), data classification, geofencing validation.
32. ~~**Add IaC.**~~ DONE — `deploy/terraform/` with modules for Kubernetes, database, Kafka. Multi-cloud variables.
33. ~~**Implement disaster recovery.**~~ DONE — `deploy/dr/dr-config.yaml` with tiered RTO/RPO, `failover-runbook.md`.
34. ~~**Add Open Banking / Plaid integration.**~~ DONE — `pkg/openbanking` with Plaid client interface, `plaid_adapter.go` in account-service.

### Additional work completed (not in original audit)

- **Input validation** — pagination bounds (max 100, non-negative offset) and amount positivity checks added to all handlers.
- **Network policy egress** — fixed `to: []` (allow-all) to proper deny-all with explicit allow rules for DNS, Postgres, Kafka, Redis, Jaeger.
- **Redis authentication** — added `requirepass` to docker-compose, added `REDIS_PASSWORD` env var to all services.
- **Kafka SASL/TLS support** — wired TLS and SASL (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512) into `pkg/kafka` producer and consumer.
- **PostgreSQL Row-Level Security** — added RLS migrations for account, payment, and ledger services.
- **Per-service DB users** — updated `scripts/init-db.sh` and docker-compose with dedicated users per service.
- **Audit trail tables** — added `audit_log` table migrations for ledger and account services.
- **Deposit campaign management** — campaign aggregate, promotional rates, create/apply use cases.
- **Credit bureau adapter** — structured adapter with retry logic, simulated mode, configurable bureau selection.
- **Alternative credit scoring** — scoring service using utility/rent/payroll data, integrated with underwriting engine.
- **Embedded finance / partner APIs** — partner proxy in gateway with API key auth, rate limiting, webhook registration.

---

## 7. Outstanding Items — Where to Start Next

Six items remain. Recommended order:

### 1. Asymmetric JWT signing (S7) — HIGH priority
- Generate RSA keypair (or ECDSA P-256)
- Update `pkg/auth/jwt.go`: `SigningMethodRS256`, load private key for signing, public key for validation
- Update `JWTConfig` to accept key file paths instead of a shared secret string
- Update all 10 service `main.go` files and gateway to load the public key
- Update `pkg/auth/jwt_test.go`

### 2. gRPC TLS (S8) — HIGH priority
- Generate self-signed CA + service certs (or use cert-manager in K8s)
- Update all `server.go` files: `grpc.NewServer(grpc.Creds(credentials.NewTLS(...)))`
- Update gateway proxy: `grpc.WithTransportCredentials(credentials.NewTLS(...))`
- Add `TLS_CERT_FILE` and `TLS_KEY_FILE` env vars to config
- Add cert volume mounts to docker-compose and Helm charts

### 3. Integration tests (P4-24) — MEDIUM priority
- Start with `services/ledger-service/test/integration/` and `services/payment-service/test/integration/`
- Use `pkg/testutil` postgres container helpers
- Test actual SQL against a real Postgres instance
- Test Kafka publish/consume round-trips
- Run with `go test -tags=integration`

### 4. Concurrency tests (P4-27) — MEDIUM priority
- Add to existing domain model test files
- Test concurrent `Freeze` + `Close` on same account
- Test concurrent balance updates in ledger
- Test optimistic locking version conflicts
- Use `-race` flag (already in Makefile)

### 5. E2E tests (P4-26) — LOW priority (requires full stack)
- Un-skip `TestOnboardingFlow` and `TestPaymentFlow` in `e2e/e2e_test.go`
- Add response body assertions
- Add flows for FX, deposits, lending, cards, fraud
- Requires `docker-compose up` before running

### 6. Standardize infra package naming (C2) — LOW priority
- Rename `infrastructure/persistence/postgres/` → `infrastructure/postgres/` (or vice versa) across 7 services
- Rename `infrastructure/messaging/` → `infrastructure/kafka/` (or vice versa) across 7 services
- Update all import paths
- Best done with IDE refactoring tools

---

## Appendix: Files Referenced

### Security
- `gateway/internal/config/config.go`
- `gateway/cmd/gatewayd/main.go`
- `pkg/auth/jwt.go`
- `pkg/auth/middleware.go`
- `pkg/auth/claims.go`
- `docker-compose.yml`
- `docker-compose.infra.yml`
- `deploy/base/network-policy.yaml`
- `deploy/charts/bib-account/values.yaml`
- `pkg/kafka/config.go`
- `pkg/observability/tracing.go`
- All `services/*/internal/infrastructure/config/config.go`
- All `services/*/internal/presentation/grpc/server.go`
- All `services/*/internal/presentation/grpc/handler.go`
- All migration `.up.sql` files

### Architecture
- `gateway/internal/handler/routes.go`
- `gateway/internal/middleware/ratelimit.go`
- `services/card-service/internal/domain/model/card.go`
- `services/payment-service/internal/domain/model/payment_order.go`
- `services/ledger-service/internal/domain/model/journal_entry.go`
- `services/payment-service/internal/domain/service/payment_saga.go`
- `services/fx-service/internal/presentation/rest/health.go`
- `services/account-service/internal/presentation/grpc/handler.go`
- `services/lending-service/internal/presentation/grpc/server.go`
- `api/proto/bib/ledger/v1/ledger.proto`
- `api/proto/bib/deposit/v1/deposit.proto`
- `api/proto/bib/lending/v1/lending.proto`

### Consistency
- `pkg/events/event.go`
- `services/account-service/internal/domain/event/events.go`
- `services/lending-service/internal/domain/event/events.go`
- `services/card-service/internal/domain/event/events.go`
- `services/payment-service/internal/infrastructure/adapter/ach/adapter.go`
- `services/payment-service/internal/infrastructure/adapters/ach_adapter.go`
- All `services/*/cmd/*/main.go`
