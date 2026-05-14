-- Update existing server database for idempotency response replay.
-- Safe to run multiple times in DBeaver or psql.

BEGIN;

ALTER TABLE idempotency_keys
ADD COLUMN IF NOT EXISTS response_status INTEGER;

ALTER TABLE idempotency_keys
ADD COLUMN IF NOT EXISTS response_content_type TEXT;

ALTER TABLE idempotency_keys
ADD COLUMN IF NOT EXISTS response_body BYTEA;

ALTER TABLE idempotency_keys
ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_completed_at
ON idempotency_keys(completed_at);

COMMIT;

-- Optional check after applying:
-- SELECT column_name, data_type, is_nullable
-- FROM information_schema.columns
-- WHERE table_name = 'idempotency_keys'
-- ORDER BY ordinal_position;
