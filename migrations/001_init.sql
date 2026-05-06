CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_number VARCHAR(32) NOT NULL UNIQUE,
    balance NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT accounts_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT accounts_currency_rub CHECK (currency = 'RUB')
);

CREATE TABLE IF NOT EXISTS cards (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    encrypted_number BYTEA NOT NULL,
    encrypted_expiry BYTEA NOT NULL,
    cvv_hash TEXT NOT NULL,
    number_hmac TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT cards_number_hmac_unique UNIQUE (number_hmac)
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

CREATE TABLE IF NOT EXISTS mfa_codes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose VARCHAR(50) NOT NULL,
    operation_hash TEXT NOT NULL DEFAULT '',
    code_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

ALTER TABLE users
ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE mfa_codes
ADD COLUMN IF NOT EXISTS operation_hash TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_accounts_user_id
ON accounts(user_id);

CREATE INDEX IF NOT EXISTS idx_cards_user_id
ON cards(user_id);

CREATE INDEX IF NOT EXISTS idx_cards_account_id
ON cards(account_id);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id
ON transactions(user_id);

CREATE INDEX IF NOT EXISTS idx_transactions_created_at
ON transactions(created_at);

CREATE INDEX IF NOT EXISTS idx_credits_user_id
ON credits(user_id);

CREATE INDEX IF NOT EXISTS idx_payment_schedules_credit_id
ON payment_schedules(credit_id);

CREATE INDEX IF NOT EXISTS idx_payment_schedules_payment_date
ON payment_schedules(payment_date);

CREATE INDEX IF NOT EXISTS idx_payment_schedules_status_payment_date
ON payment_schedules(status, payment_date);

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at
ON revoked_tokens(expires_at);

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user_id
ON revoked_tokens(user_id);

CREATE INDEX IF NOT EXISTS idx_mfa_codes_user_id_purpose
ON mfa_codes(user_id, purpose);

CREATE INDEX IF NOT EXISTS idx_mfa_codes_expires_at
ON mfa_codes(expires_at);

CREATE INDEX IF NOT EXISTS idx_mfa_codes_user_purpose_operation
ON mfa_codes(user_id, purpose, operation_hash);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id
ON user_sessions(user_id);

CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at
ON user_sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_user_sessions_active
ON user_sessions(user_id, expires_at)
WHERE revoked_at IS NULL;
