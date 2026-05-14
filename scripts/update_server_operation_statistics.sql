-- Update existing server database for account/card operation statistics.
-- Safe to run multiple times in DBeaver or psql.

BEGIN;

ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS from_card_id BIGINT;

ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS to_card_id BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'transactions_from_card_id_fkey'
    ) THEN
        ALTER TABLE transactions
        ADD CONSTRAINT transactions_from_card_id_fkey
        FOREIGN KEY (from_card_id) REFERENCES cards(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'transactions_to_card_id_fkey'
    ) THEN
        ALTER TABLE transactions
        ADD CONSTRAINT transactions_to_card_id_fkey
        FOREIGN KEY (to_card_id) REFERENCES cards(id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_transactions_from_account_id
ON transactions(from_account_id);

CREATE INDEX IF NOT EXISTS idx_transactions_to_account_id
ON transactions(to_account_id);

CREATE INDEX IF NOT EXISTS idx_transactions_from_card_id
ON transactions(from_card_id);

CREATE INDEX IF NOT EXISTS idx_transactions_to_card_id
ON transactions(to_card_id);

-- Conservative backfill for old card payments only when an account has exactly one card.
-- If several cards are attached to the same account, historical card ownership cannot be restored reliably.
WITH single_account_cards AS (
    SELECT user_id, account_id, MIN(id) AS card_id
    FROM cards
    GROUP BY user_id, account_id
    HAVING COUNT(*) = 1
)
UPDATE transactions t
SET from_card_id = sac.card_id
FROM single_account_cards sac
WHERE t.type = 'card_payment'
  AND t.from_card_id IS NULL
  AND t.user_id = sac.user_id
  AND t.from_account_id = sac.account_id;

COMMIT;

-- Optional check after applying:
-- SELECT column_name, data_type, is_nullable
-- FROM information_schema.columns
-- WHERE table_name = 'transactions'
--   AND column_name IN ('from_card_id', 'to_card_id')
-- ORDER BY ordinal_position;
