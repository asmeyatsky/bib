CREATE TABLE IF NOT EXISTS payment_orders (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    source_account_id UUID NOT NULL,
    destination_account_id UUID,
    amount NUMERIC(19,4) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    rail VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'INITIATED',
    routing_number VARCHAR(9) NOT NULL DEFAULT '',
    external_account_number VARCHAR(34) NOT NULL DEFAULT '',
    reference VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    failure_reason TEXT NOT NULL DEFAULT '',
    initiated_at TIMESTAMPTZ NOT NULL,
    settled_at TIMESTAMPTZ,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_tenant ON payment_orders (tenant_id);
CREATE INDEX idx_payments_source ON payment_orders (source_account_id);
CREATE INDEX idx_payments_status ON payment_orders (status);

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
