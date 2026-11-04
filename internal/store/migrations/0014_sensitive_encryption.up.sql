-- 0014_sensitive_encryption.up.sql: enforce sensitive column encryption metadata
ALTER TABLE channels ADD COLUMN IF NOT EXISTS encryption_key_version VARCHAR(32) DEFAULT 'v1';
ALTER TABLE channels ADD COLUMN IF NOT EXISTS encrypted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_channels_encryption_version ON channels(encryption_key_version);
