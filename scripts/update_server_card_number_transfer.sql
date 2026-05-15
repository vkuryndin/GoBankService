-- Enable/verify fast lookup for card transfer by recipient card number.
-- The project already stores a secure HMAC fingerprint of the card number in cards.number_hmac.
-- This script is safe to run multiple times.

BEGIN;

CREATE INDEX IF NOT EXISTS idx_cards_number_hmac_lookup
ON cards(number_hmac);

COMMIT;

-- Optional check:
-- SELECT indexname, indexdef
-- FROM pg_indexes
-- WHERE tablename = 'cards' AND indexname IN ('cards_number_hmac_unique', 'idx_cards_number_hmac_lookup')
-- ORDER BY indexname;
