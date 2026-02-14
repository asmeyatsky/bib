#!/bin/bash
set -e

# Create schema-per-service databases for local development
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
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
EOSQL
