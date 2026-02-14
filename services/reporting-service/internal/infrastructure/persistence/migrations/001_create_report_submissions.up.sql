CREATE TABLE IF NOT EXISTS report_submissions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    report_type VARCHAR(20) NOT NULL,
    reporting_period VARCHAR(10) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    xbrl_content TEXT NOT NULL DEFAULT '',
    generated_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,
    validation_errors JSONB NOT NULL DEFAULT '[]',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reports_tenant ON report_submissions (tenant_id);
CREATE INDEX idx_reports_type ON report_submissions (report_type);
CREATE INDEX idx_reports_period ON report_submissions (reporting_period);

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
