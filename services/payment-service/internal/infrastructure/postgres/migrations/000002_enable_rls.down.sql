DROP POLICY IF EXISTS tenant_isolation ON payment_orders;
ALTER TABLE payment_orders DISABLE ROW LEVEL SECURITY;
