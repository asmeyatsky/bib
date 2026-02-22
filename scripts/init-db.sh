#!/bin/bash
set -e

# Create schema-per-service databases and dedicated users for local development.
# Each service gets its own database and a least-privilege user that can only
# access its own database.

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create databases
    CREATE DATABASE bib_ledger;
    CREATE DATABASE bib_account;
    CREATE DATABASE bib_deposit;
    CREATE DATABASE bib_identity;
    CREATE DATABASE bib_payment;
    CREATE DATABASE bib_fx;
    CREATE DATABASE bib_lending;
    CREATE DATABASE bib_fraud;
    CREATE DATABASE bib_card;
    CREATE DATABASE bib_reporting;

    -- Create per-service users with limited privileges
    CREATE USER bib_ledger_user   WITH PASSWORD 'ledger_dev_password';
    CREATE USER bib_account_user  WITH PASSWORD 'account_dev_password';
    CREATE USER bib_deposit_user  WITH PASSWORD 'deposit_dev_password';
    CREATE USER bib_identity_user WITH PASSWORD 'identity_dev_password';
    CREATE USER bib_payment_user  WITH PASSWORD 'payment_dev_password';
    CREATE USER bib_fx_user       WITH PASSWORD 'fx_dev_password';
    CREATE USER bib_lending_user  WITH PASSWORD 'lending_dev_password';
    CREATE USER bib_fraud_user    WITH PASSWORD 'fraud_dev_password';
    CREATE USER bib_card_user     WITH PASSWORD 'card_dev_password';
    CREATE USER bib_reporting_user WITH PASSWORD 'reporting_dev_password';
EOSQL

# Grant per-service privileges on each database.
# Each user gets CONNECT + full schema usage on its own database only.
grant_service_access() {
    local db=$1
    local user=$2
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db" <<-EOSQL
        GRANT CONNECT ON DATABASE $db TO $user;
        GRANT USAGE, CREATE ON SCHEMA public TO $user;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO $user;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO $user;
EOSQL
}

grant_service_access bib_ledger   bib_ledger_user
grant_service_access bib_account  bib_account_user
grant_service_access bib_deposit  bib_deposit_user
grant_service_access bib_identity bib_identity_user
grant_service_access bib_payment  bib_payment_user
grant_service_access bib_fx       bib_fx_user
grant_service_access bib_lending  bib_lending_user
grant_service_access bib_fraud    bib_fraud_user
grant_service_access bib_card     bib_card_user
grant_service_access bib_reporting bib_reporting_user
