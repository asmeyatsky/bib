CREATE TABLE IF NOT EXISTS exchange_rates (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    base_currency VARCHAR(3) NOT NULL,
    quote_currency VARCHAR(3) NOT NULL,
    rate NUMERIC(19,10) NOT NULL,
    inverse_rate NUMERIC(19,10) NOT NULL,
    provider VARCHAR(100) NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_exchange_rates_pair ON exchange_rates (base_currency, quote_currency);
CREATE INDEX idx_exchange_rates_effective ON exchange_rates (effective_at);
CREATE UNIQUE INDEX idx_exchange_rates_latest ON exchange_rates (tenant_id, base_currency, quote_currency, effective_at);

CREATE TABLE IF NOT EXISTS revaluation_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    functional_currency VARCHAR(3) NOT NULL,
    as_of_date TIMESTAMPTZ NOT NULL,
    total_gain_loss NUMERIC(19,4) NOT NULL,
    accounts_processed INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
