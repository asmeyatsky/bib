CREATE TABLE IF NOT EXISTS cards (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    card_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    last_four VARCHAR(4) NOT NULL,
    expiry_month VARCHAR(2) NOT NULL,
    expiry_year VARCHAR(4) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    daily_limit NUMERIC(19,4) NOT NULL,
    monthly_limit NUMERIC(19,4) NOT NULL,
    daily_spent NUMERIC(19,4) NOT NULL DEFAULT 0,
    monthly_spent NUMERIC(19,4) NOT NULL DEFAULT 0,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cards_tenant ON cards (tenant_id);
CREATE INDEX idx_cards_account ON cards (account_id);
CREATE INDEX idx_cards_status ON cards (status);

CREATE TABLE IF NOT EXISTS card_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID NOT NULL REFERENCES cards(id),
    amount NUMERIC(19,4) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    merchant_name VARCHAR(255) NOT NULL,
    merchant_category VARCHAR(10) NOT NULL DEFAULT '',
    auth_code VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_card_txns_card ON card_transactions (card_id);

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
