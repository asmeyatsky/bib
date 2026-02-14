-- Deposit products table
CREATE TABLE IF NOT EXISTS deposit_products (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    term_days INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deposit_products_tenant ON deposit_products (tenant_id);

-- Interest tiers (child of deposit products)
CREATE TABLE IF NOT EXISTS interest_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES deposit_products(id),
    min_balance NUMERIC(19,4) NOT NULL,
    max_balance NUMERIC(19,4) NOT NULL,
    rate_bps INT NOT NULL,
    seq_num INT NOT NULL
);

CREATE INDEX idx_tiers_product ON interest_tiers (product_id);

-- Deposit positions (customer deposit holdings)
CREATE TABLE IF NOT EXISTS deposit_positions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    product_id UUID NOT NULL REFERENCES deposit_products(id),
    principal NUMERIC(19,4) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    accrued_interest NUMERIC(19,4) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    opened_at TIMESTAMPTZ NOT NULL,
    maturity_date TIMESTAMPTZ,
    last_accrual_date TIMESTAMPTZ NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_positions_tenant ON deposit_positions (tenant_id);
CREATE INDEX idx_positions_account ON deposit_positions (account_id);
CREATE INDEX idx_positions_status ON deposit_positions (status);

-- Outbox table for transactional outbox pattern
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
