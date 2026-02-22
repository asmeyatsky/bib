-- Enable Row-Level Security for tenant isolation on journal_entries
ALTER TABLE journal_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON journal_entries
    USING (tenant_id::text = current_setting('app.tenant_id'));

-- Enable RLS on fiscal_periods
ALTER TABLE fiscal_periods ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON fiscal_periods
    USING (tenant_id::text = current_setting('app.tenant_id'));
