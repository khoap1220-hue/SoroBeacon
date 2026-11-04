-- 0015_alert_statistics_summary.up.sql
CREATE TABLE IF NOT EXISTS alert_statistics_summary (
    workspace_id VARCHAR(64) NOT NULL,
    monitor_id VARCHAR(64) NOT NULL DEFAULT '',
    rule_id VARCHAR(64) NOT NULL DEFAULT '',
    severity VARCHAR(32) NOT NULL DEFAULT 'info',
    state VARCHAR(32) NOT NULL DEFAULT 'firing',
    alert_count BIGINT NOT NULL DEFAULT 0,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, monitor_id, rule_id, severity, state)
);

CREATE INDEX IF NOT EXISTS idx_alert_stats_ws ON alert_statistics_summary(workspace_id);
