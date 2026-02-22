-- Enable Row-Level Security for tenant isolation on customer_accounts
ALTER TABLE customer_accounts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON customer_accounts
    USING (tenant_id::text = current_setting('app.tenant_id'));

-- RLS on account_holders via the parent account's tenant
-- account_holders does not have a direct tenant_id, so we use a join policy
ALTER TABLE account_holders ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON account_holders
    USING (account_id IN (
        SELECT id FROM customer_accounts
        WHERE tenant_id::text = current_setting('app.tenant_id')
    ));
