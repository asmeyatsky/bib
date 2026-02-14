DROP INDEX IF EXISTS idx_outbox_unpublished;
DROP TABLE IF EXISTS outbox;

DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_source;
DROP INDEX IF EXISTS idx_payments_tenant;
DROP TABLE IF EXISTS payment_orders;
