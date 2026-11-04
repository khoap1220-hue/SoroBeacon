-- 0014_sensitive_encryption.down.sql
DROP INDEX IF EXISTS idx_channels_encryption_version;
ALTER TABLE channels DROP COLUMN IF EXISTS encrypted_at;
ALTER TABLE channels DROP COLUMN IF EXISTS encryption_key_version;
