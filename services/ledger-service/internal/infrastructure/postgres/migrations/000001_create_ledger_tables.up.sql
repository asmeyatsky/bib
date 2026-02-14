-- Journal entries table
CREATE TABLE IF NOT EXISTS journal_entries (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    effective_date  TIMESTAMPTZ NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    description     TEXT NOT NULL DEFAULT '',
    reference       VARCHAR(255) NOT NULL DEFAULT '',
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_journal_entries_tenant ON journal_entries (tenant_id);
CREATE INDEX idx_journal_entries_effective_date ON journal_entries (effective_date);
CREATE INDEX idx_journal_entries_status ON journal_entries (status);
CREATE INDEX idx_journal_entries_reference ON journal_entries (reference);

-- Posting pairs (child of journal entries)
CREATE TABLE IF NOT EXISTS posting_pairs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id        UUID NOT NULL REFERENCES journal_entries(id),
    debit_account   VARCHAR(10) NOT NULL,
    credit_account  VARCHAR(10) NOT NULL,
    amount          NUMERIC(19,4) NOT NULL,
    currency        VARCHAR(3) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    seq_num         INT NOT NULL,
    CONSTRAINT chk_positive_amount CHECK (amount > 0),
    CONSTRAINT chk_different_accounts CHECK (debit_account != credit_account)
);

CREATE INDEX idx_posting_pairs_entry ON posting_pairs (entry_id);
CREATE INDEX idx_posting_pairs_debit ON posting_pairs (debit_account);
CREATE INDEX idx_posting_pairs_credit ON posting_pairs (credit_account);

-- Account balances (materialized running balances)
CREATE TABLE IF NOT EXISTS account_balances (
    account_code    VARCHAR(10) NOT NULL,
    currency        VARCHAR(3) NOT NULL,
    balance         NUMERIC(19,4) NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_code, currency)
);

-- Fiscal periods
CREATE TABLE IF NOT EXISTS fiscal_periods (
    tenant_id   UUID NOT NULL,
    year        INT NOT NULL,
    month       INT NOT NULL,
    status      VARCHAR(10) NOT NULL DEFAULT 'OPEN',
    closed_at   TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, year, month)
);

-- Outbox table for transactional outbox pattern
CREATE TABLE IF NOT EXISTS outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id    UUID NOT NULL,
    aggregate_type  VARCHAR(100) NOT NULL,
    event_type      VARCHAR(100) NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;
