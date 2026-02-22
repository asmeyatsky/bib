-- Enable Row-Level Security for tenant isolation on payment_orders
ALTER TABLE payment_orders ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON payment_orders
    USING (tenant_id::text = current_setting('app.tenant_id'));
