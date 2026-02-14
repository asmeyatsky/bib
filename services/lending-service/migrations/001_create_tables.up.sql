-- Lending Service Schema
-- Database: bib_lending

BEGIN;

-- ---------------------------------------------------------------------------
-- loan_applications (Loan Origination System)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS loan_applications (
    id                UUID PRIMARY KEY,
    tenant_id         UUID        NOT NULL,
    applicant_id      UUID        NOT NULL,
    requested_amount  NUMERIC(20,4) NOT NULL CHECK (requested_amount > 0),
    currency          VARCHAR(3)  NOT NULL,
    term_months       INT         NOT NULL CHECK (term_months > 0),
    purpose           TEXT        NOT NULL DEFAULT '',
    status            VARCHAR(20) NOT NULL DEFAULT 'SUBMITTED',
    decision_reason   TEXT        NOT NULL DEFAULT '',
    credit_score      VARCHAR(10) NOT NULL DEFAULT '',
    version           INT         NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loan_applications_tenant     ON loan_applications (tenant_id);
CREATE INDEX idx_loan_applications_applicant  ON loan_applications (tenant_id, applicant_id);
CREATE INDEX idx_loan_applications_status     ON loan_applications (tenant_id, status);

-- ---------------------------------------------------------------------------
-- loans (Loan Servicing System)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS loans (
    id                  UUID PRIMARY KEY,
    tenant_id           UUID           NOT NULL,
    application_id      UUID           NOT NULL REFERENCES loan_applications(id),
    borrower_account_id UUID           NOT NULL,
    principal           NUMERIC(20,4)  NOT NULL CHECK (principal > 0),
    currency            VARCHAR(3)     NOT NULL,
    interest_rate_bps   INT            NOT NULL CHECK (interest_rate_bps >= 0),
    term_months         INT            NOT NULL CHECK (term_months > 0),
    status              VARCHAR(20)    NOT NULL DEFAULT 'ACTIVE',
    outstanding_balance NUMERIC(20,4)  NOT NULL,
    next_payment_due    TIMESTAMPTZ,
    version             INT            NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loans_tenant       ON loans (tenant_id);
CREATE INDEX idx_loans_application  ON loans (tenant_id, application_id);
CREATE INDEX idx_loans_borrower     ON loans (tenant_id, borrower_account_id);
CREATE INDEX idx_loans_status       ON loans (tenant_id, status);

-- ---------------------------------------------------------------------------
-- amortization_entries
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS amortization_entries (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_id           UUID           NOT NULL REFERENCES loans(id),
    period            INT            NOT NULL,
    due_date          TIMESTAMPTZ    NOT NULL,
    principal         NUMERIC(20,4)  NOT NULL,
    interest          NUMERIC(20,4)  NOT NULL,
    total             NUMERIC(20,4)  NOT NULL,
    remaining_balance NUMERIC(20,4)  NOT NULL,

    UNIQUE (loan_id, period)
);

CREATE INDEX idx_amortization_loan ON amortization_entries (loan_id);

-- ---------------------------------------------------------------------------
-- collection_cases
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS collection_cases (
    id          UUID PRIMARY KEY,
    loan_id     UUID        NOT NULL REFERENCES loans(id),
    tenant_id   UUID        NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    assigned_to VARCHAR(255) NOT NULL DEFAULT '',
    notes       JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_collection_cases_loan   ON collection_cases (tenant_id, loan_id);
CREATE INDEX idx_collection_cases_status ON collection_cases (tenant_id, status);

-- ---------------------------------------------------------------------------
-- outbox (transactional outbox for event publishing)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS outbox (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id  UUID         NOT NULL,
    tenant_id     UUID         NOT NULL,
    event_type    VARCHAR(100) NOT NULL,
    payload       JSONB        NOT NULL,
    published     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbox_unpublished ON outbox (published, created_at) WHERE NOT published;

COMMIT;
