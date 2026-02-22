DROP POLICY IF EXISTS tenant_isolation ON fiscal_periods;
ALTER TABLE fiscal_periods DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON journal_entries;
ALTER TABLE journal_entries DISABLE ROW LEVEL SECURITY;
