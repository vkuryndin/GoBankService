CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_number VARCHAR(32) NOT NULL UNIQUE,
    balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT accounts_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT accounts_currency_rub CHECK (currency = 'RUB'),
    CONSTRAINT accounts_status_check CHECK (status IN ('active', 'closed'))
);

CREATE TABLE IF NOT EXISTS cards (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    encrypted_number BYTEA NOT NULL,
    encrypted_expiry BYTEA NOT NULL,
    cvv_hash TEXT NOT NULL,
    number_hmac TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT cards_number_hmac_unique UNIQUE (number_hmac),
    CONSTRAINT cards_status_check CHECK (status IN ('active', 'closed'))
);

CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    to_account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    amount NUMERIC(14,2) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    type VARCHAR(30) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'completed',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT transactions_amount_positive CHECK (amount > 0),
    CONSTRAINT transactions_currency_rub CHECK (currency = 'RUB'),
    CONSTRAINT transactions_type_check CHECK (
        type IN (
            'deposit',
            'withdraw',
            'transfer',
            'card_payment',
            'credit_payment',
            'credit_issue',
            'penalty'
        )
    ),
    CONSTRAINT transactions_status_check CHECK (
        status IN ('pending', 'completed', 'failed')
    )
);

CREATE TABLE IF NOT EXISTS credits (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    principal_amount NUMERIC(14,2) NOT NULL,
    interest_rate NUMERIC(5,2) NOT NULL,
    term_months INT NOT NULL,
    monthly_payment NUMERIC(14,2) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT credits_principal_positive CHECK (principal_amount > 0),
    CONSTRAINT credits_interest_non_negative CHECK (interest_rate >= 0),
    CONSTRAINT credits_term_positive CHECK (term_months > 0),
    CONSTRAINT credits_monthly_payment_positive CHECK (monthly_payment > 0),
    CONSTRAINT credits_status_check CHECK (
        status IN ('active', 'closed', 'overdue')
    )
);

CREATE TABLE IF NOT EXISTS payment_schedules (
    id BIGSERIAL PRIMARY KEY,
    credit_id BIGINT NOT NULL REFERENCES credits(id) ON DELETE CASCADE,
    payment_date DATE NOT NULL,
    amount NUMERIC(14,2) NOT NULL,
    penalty_amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT payment_schedules_amount_positive CHECK (amount > 0),
    CONSTRAINT payment_schedules_penalty_non_negative CHECK (penalty_amount >= 0),
    CONSTRAINT payment_schedules_status_check CHECK (
        status IN ('pending', 'paid', 'overdue')
    )
);

CREATE TABLE IF NOT EXISTS revoked_tokens (
    id BIGSERIAL PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at
ON revoked_tokens(expires_at);

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user_id
ON revoked_tokens(user_id);


CREATE TABLE IF NOT EXISTS mfa_codes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose VARCHAR(50) NOT NULL,
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mfa_codes_user_id_purpose
ON mfa_codes(user_id, purpose);

CREATE INDEX IF NOT EXISTS idx_mfa_codes_expires_at
ON mfa_codes(expires_at);


ALTER TABLE mfa_codes
ADD COLUMN IF NOT EXISTS operation_hash TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_mfa_codes_user_purpose_operation
ON mfa_codes(user_id, purpose, operation_hash);

CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_cards_user_id ON cards(user_id);
CREATE INDEX IF NOT EXISTS idx_cards_account_id ON cards(account_id);
CREATE INDEX IF NOT EXISTS idx_cards_user_status ON cards(user_id, status);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);
CREATE INDEX IF NOT EXISTS idx_credits_user_id ON credits(user_id);
CREATE INDEX IF NOT EXISTS idx_payment_schedules_credit_id ON payment_schedules(credit_id);
CREATE INDEX IF NOT EXISTS idx_payment_schedules_payment_date ON payment_schedules(payment_date);


ALTER TABLE users
ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';

ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_accounts_user_status
ON accounts(user_id, status);

CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id
ON user_sessions(user_id);

CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at
ON user_sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_user_sessions_active
ON user_sessions(user_id, expires_at)
WHERE revoked_at IS NULL;
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method VARCHAR(16) NOT NULL,
    path TEXT NOT NULL,
    key TEXT NOT NULL,
    request_hash TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT idempotency_keys_unique_operation UNIQUE (user_id, method, path, key)
);

ALTER TABLE idempotency_keys
ADD COLUMN IF NOT EXISTS request_hash TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_created_at
ON idempotency_keys(created_at);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_request_hash
ON idempotency_keys(request_hash);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64),
    resource_id BIGINT,
    status VARCHAR(32) NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT audit_logs_action_not_empty CHECK (action <> ''),
    CONSTRAINT audit_logs_status_check CHECK (status IN ('success', 'failed', 'blocked'))
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id
ON audit_logs(user_id);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action
ON audit_logs(action);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at
ON audit_logs(created_at);

-- Compatibility constraints for databases that were created before constraints
-- were moved into the CREATE TABLE definitions above.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'idempotency_keys') THEN
        ALTER TABLE idempotency_keys ADD COLUMN IF NOT EXISTS request_hash TEXT NOT NULL DEFAULT '';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_email_not_empty') THEN
        ALTER TABLE users ADD CONSTRAINT users_email_not_empty CHECK (email <> '');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_username_not_empty') THEN
        ALTER TABLE users ADD CONSTRAINT users_username_not_empty CHECK (username <> '');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_password_hash_not_empty') THEN
        ALTER TABLE users ADD CONSTRAINT users_password_hash_not_empty CHECK (password_hash <> '');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'accounts_balance_non_negative') THEN
        ALTER TABLE accounts ADD CONSTRAINT accounts_balance_non_negative CHECK (balance >= 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'accounts_currency_rub') THEN
        ALTER TABLE accounts ADD CONSTRAINT accounts_currency_rub CHECK (currency = 'RUB');
    END IF;

    ALTER TABLE accounts ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
    ALTER TABLE accounts ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'accounts_status_check') THEN
        ALTER TABLE accounts ADD CONSTRAINT accounts_status_check CHECK (status IN ('active', 'closed'));
    END IF;


    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'cards') THEN
        ALTER TABLE cards ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
        ALTER TABLE cards ADD COLUMN IF NOT EXISTS closed_at TIMESTAMPTZ;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'cards_status_check') THEN
        ALTER TABLE cards ADD CONSTRAINT cards_status_check CHECK (status IN ('active', 'closed'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'transactions_amount_positive') THEN
        ALTER TABLE transactions ADD CONSTRAINT transactions_amount_positive CHECK (amount > 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'transactions_currency_rub') THEN
        ALTER TABLE transactions ADD CONSTRAINT transactions_currency_rub CHECK (currency = 'RUB');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'transactions_type_check') THEN
        ALTER TABLE transactions ADD CONSTRAINT transactions_type_check CHECK (
            type IN ('deposit', 'withdraw', 'transfer', 'card_payment', 'credit_payment', 'credit_issue', 'penalty')
        );
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'transactions_status_check') THEN
        ALTER TABLE transactions ADD CONSTRAINT transactions_status_check CHECK (status IN ('pending', 'completed', 'failed'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'credits_principal_positive') THEN
        ALTER TABLE credits ADD CONSTRAINT credits_principal_positive CHECK (principal_amount > 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'credits_interest_non_negative') THEN
        ALTER TABLE credits ADD CONSTRAINT credits_interest_non_negative CHECK (interest_rate >= 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'credits_term_positive') THEN
        ALTER TABLE credits ADD CONSTRAINT credits_term_positive CHECK (term_months > 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'credits_monthly_payment_positive') THEN
        ALTER TABLE credits ADD CONSTRAINT credits_monthly_payment_positive CHECK (monthly_payment > 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'credits_status_check') THEN
        ALTER TABLE credits ADD CONSTRAINT credits_status_check CHECK (status IN ('active', 'closed', 'overdue'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payment_schedules_amount_positive') THEN
        ALTER TABLE payment_schedules ADD CONSTRAINT payment_schedules_amount_positive CHECK (amount > 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payment_schedules_penalty_non_negative') THEN
        ALTER TABLE payment_schedules ADD CONSTRAINT payment_schedules_penalty_non_negative CHECK (penalty_amount >= 0);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payment_schedules_status_check') THEN
        ALTER TABLE payment_schedules ADD CONSTRAINT payment_schedules_status_check CHECK (status IN ('pending', 'paid', 'overdue'));
    END IF;
END $$;
