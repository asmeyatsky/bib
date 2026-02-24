CREATE TABLE IF NOT EXISTS loan_applications (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT        NOT NULL,
    applicant_id    TEXT        NOT NULL,
    requested_amount NUMERIC    NOT NULL,
    currency        TEXT        NOT NULL,
    term_months     INT         NOT NULL,
    purpose         TEXT        NOT NULL DEFAULT '',
    status          TEXT        NOT NULL,
    decision_reason TEXT        NOT NULL DEFAULT '',
    credit_score    TEXT        NOT NULL DEFAULT '',
    version         INT         NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS loans (
    id                   TEXT PRIMARY KEY,
    tenant_id            TEXT        NOT NULL,
    application_id       TEXT        NOT NULL,
    borrower_account_id  TEXT        NOT NULL,
    principal            NUMERIC     NOT NULL,
    currency             TEXT        NOT NULL,
    interest_rate_bps    INT         NOT NULL,
    term_months          INT         NOT NULL,
    status               TEXT        NOT NULL,
    outstanding_balance  NUMERIC     NOT NULL,
    next_payment_due     TIMESTAMPTZ,
    version              INT         NOT NULL DEFAULT 1,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS amortization_entries (
    loan_id            TEXT    NOT NULL REFERENCES loans(id),
    period             INT     NOT NULL,
    due_date           TIMESTAMPTZ NOT NULL,
    principal          NUMERIC NOT NULL,
    interest           NUMERIC NOT NULL,
    total              NUMERIC NOT NULL,
    remaining_balance  NUMERIC NOT NULL,
    PRIMARY KEY (loan_id, period)
);

CREATE TABLE IF NOT EXISTS collection_cases (
    id          TEXT PRIMARY KEY,
    loan_id     TEXT        NOT NULL REFERENCES loans(id),
    tenant_id   TEXT        NOT NULL,
    status      TEXT        NOT NULL,
    assigned_to TEXT        NOT NULL DEFAULT '',
    notes       JSONB       NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
