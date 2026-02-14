DROP INDEX IF EXISTS idx_outbox_unpublished;
DROP TABLE IF EXISTS outbox;

DROP INDEX IF EXISTS idx_card_txns_card;
DROP TABLE IF EXISTS card_transactions;

DROP INDEX IF EXISTS idx_cards_status;
DROP INDEX IF EXISTS idx_cards_account;
DROP INDEX IF EXISTS idx_cards_tenant;
DROP TABLE IF EXISTS cards;
