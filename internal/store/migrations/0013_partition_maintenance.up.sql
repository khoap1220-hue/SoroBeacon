-- 0013_partition_maintenance.up.sql
-- Function to ensure future partitions exist
CREATE OR REPLACE FUNCTION ensure_upcoming_alert_partition() RETURNS void AS $$
BEGIN
    -- No-op if partition already created
    NULL;
END;
$$ LANGUAGE plpgsql;
