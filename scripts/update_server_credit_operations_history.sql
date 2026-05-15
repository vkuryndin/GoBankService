-- Adds exact credit operation history support.
-- Safe to run more than once.

ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS credit_id BIGINT REFERENCES credits(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_transactions_credit_id
ON transactions(credit_id);

-- Keep transaction type constraint compatible with credit operations.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'transactions_type_check') THEN
        ALTER TABLE transactions DROP CONSTRAINT transactions_type_check;
    END IF;

    ALTER TABLE transactions
    ADD CONSTRAINT transactions_type_check CHECK (
        type IN (
            'deposit',
            'withdraw',
            'transfer',
            'card_payment',
            'card_transfer',
            'credit_payment',
            'credit_issue',
            'credit_prepayment',
            'penalty'
        )
    );
END $$;
