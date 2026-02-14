CREATE TABLE IF NOT EXISTS identity_verifications (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    applicant_first_name VARCHAR(100) NOT NULL,
    applicant_last_name VARCHAR(100) NOT NULL,
    applicant_email VARCHAR(255) NOT NULL,
    applicant_dob VARCHAR(10) NOT NULL,
    applicant_country VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS verification_checks (
    id UUID PRIMARY KEY,
    verification_id UUID NOT NULL REFERENCES identity_verifications(id),
    check_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    provider VARCHAR(50) NOT NULL DEFAULT '',
    provider_reference VARCHAR(255) NOT NULL DEFAULT '',
    completed_at TIMESTAMPTZ,
    failure_reason TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_verifications_tenant ON identity_verifications (tenant_id);
CREATE INDEX idx_checks_verification ON verification_checks (verification_id);

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
