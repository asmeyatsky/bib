-- 001_create_transaction_assessments.up.sql
-- Transaction assessments table for fraud risk scoring.

CREATE TABLE IF NOT EXISTS transaction_assessments (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    transaction_id  UUID NOT NULL,
    account_id      UUID NOT NULL,
    amount          NUMERIC(19,4) NOT NULL,
    currency        VARCHAR(3) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    risk_level      VARCHAR(20) NOT NULL DEFAULT 'LOW',
    risk_score      INTEGER NOT NULL DEFAULT 0 CHECK (risk_score >= 0 AND risk_score <= 100),
    decision        VARCHAR(20) NOT NULL DEFAULT 'APPROVE',
    assessed_at     TIMESTAMPTZ,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common query patterns.
CREATE INDEX idx_transaction_assessments_tenant_id ON transaction_assessments(tenant_id);
CREATE INDEX idx_transaction_assessments_transaction_id ON transaction_assessments(tenant_id, transaction_id);
CREATE INDEX idx_transaction_assessments_account_id ON transaction_assessments(tenant_id, account_id);
CREATE INDEX idx_transaction_assessments_risk_level ON transaction_assessments(tenant_id, risk_level);
CREATE INDEX idx_transaction_assessments_decision ON transaction_assessments(tenant_id, decision);
CREATE INDEX idx_transaction_assessments_assessed_at ON transaction_assessments(assessed_at DESC);

-- Unique constraint: one assessment per transaction per tenant.
CREATE UNIQUE INDEX idx_transaction_assessments_unique_txn ON transaction_assessments(tenant_id, transaction_id);
