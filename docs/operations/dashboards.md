# SoroBeacon Operator Dashboards Runbook

This directory contains pre-configured Grafana dashboards for monitoring SoroBeacon production deployments.

## Dashboards
- `sorobeacon-overview.json`: Core operational view tracking ingestion throughput, active firing alerts by severity, poller latency P99, and PostgreSQL connection pool saturation.

## Import Instructions
1. Navigate to Grafana > Dashboards > Import.
2. Upload `deploy/dashboards/sorobeacon-overview.json`.
3. Select your Prometheus data source.
