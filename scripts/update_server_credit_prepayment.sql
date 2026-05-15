-- Adds support for credit prepayment transaction type.
-- Safe to run more than once.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'credits_principal_positive') THEN
        ALTER TABLE credits DROP CONSTRAINT credits_principal_positive;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'credits_principal_non_negative') THEN
        ALTER TABLE credits
        ADD CONSTRAINT credits_principal_non_negative CHECK (principal_amount >= 0);
    END IF;

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
