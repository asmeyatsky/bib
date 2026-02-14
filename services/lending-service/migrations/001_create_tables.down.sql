BEGIN;

DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS collection_cases;
DROP TABLE IF EXISTS amortization_entries;
DROP TABLE IF EXISTS loans;
DROP TABLE IF EXISTS loan_applications;

COMMIT;
