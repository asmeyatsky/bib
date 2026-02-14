CREATE TABLE IF NOT EXISTS customer_accounts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    account_number VARCHAR(20) NOT NULL UNIQUE,
    account_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    currency VARCHAR(3) NOT NULL,
    ledger_account_code VARCHAR(10) NOT NULL DEFAULT '',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounts_tenant ON customer_accounts (tenant_id);
CREATE INDEX idx_accounts_status ON customer_accounts (status);

CREATE TABLE IF NOT EXISTS account_holders (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES customer_accounts(id),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    identity_verification_id UUID,
    UNIQUE(account_id)
);

CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);
CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
