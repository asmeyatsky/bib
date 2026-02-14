-- 003_create_outbox.up.sql
-- Transactional outbox table for reliable event publishing.

CREATE TABLE IF NOT EXISTS outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    aggregate_id    UUID NOT NULL,
    aggregate_type  VARCHAR(100) NOT NULL,
    event_type      VARCHAR(100) NOT NULL,
    payload         JSONB NOT NULL,
    published       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox(published, created_at) WHERE published = FALSE;
CREATE INDEX idx_outbox_aggregate ON outbox(aggregate_type, aggregate_id);
CREATE INDEX idx_outbox_tenant_id ON outbox(tenant_id);
