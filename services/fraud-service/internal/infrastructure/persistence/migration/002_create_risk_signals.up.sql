-- 002_create_risk_signals.up.sql
-- Risk signals table stores individual signals detected for each assessment.

CREATE TABLE IF NOT EXISTS risk_signals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id   UUID NOT NULL REFERENCES transaction_assessments(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL,
    signal          VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_risk_signals_assessment_id ON risk_signals(assessment_id);
CREATE INDEX idx_risk_signals_tenant_id ON risk_signals(tenant_id);
CREATE INDEX idx_risk_signals_signal ON risk_signals(signal);
