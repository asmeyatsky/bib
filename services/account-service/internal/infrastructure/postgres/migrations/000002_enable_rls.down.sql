DROP POLICY IF EXISTS tenant_isolation ON account_holders;
ALTER TABLE account_holders DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON customer_accounts;
ALTER TABLE customer_accounts DISABLE ROW LEVEL SECURITY;
