# Bank-in-a-Box Disaster Recovery Failover Runbook

## Overview

This runbook documents the procedures for executing a disaster recovery failover
from the primary region (us-east-1) to the DR region (us-west-2). Follow these
steps in order during a declared disaster event.

## Prerequisites

- Access to AWS Console and CLI with appropriate IAM permissions
- Access to the Kubernetes clusters in both regions
- PagerDuty incident created and war room established
- DR configuration reviewed: `deploy/dr/dr-config.yaml`

## Decision Criteria

Initiate failover when ANY of the following conditions persist for more than
15 minutes:

1. Primary region is completely unavailable
2. Primary database is unreachable and cannot be recovered
3. Network connectivity to primary region is lost
4. Primary region health checks fail beyond the configured threshold

## Failover Procedure

### Phase 1: Assessment (5 minutes)

1. Confirm the outage is region-wide and not a transient issue
2. Check AWS Health Dashboard for known incidents
3. Verify DR region health: `kubectl --context dr get nodes`
4. Check replication lag for all databases
5. Document the current replication state and last known good timestamp

### Phase 2: DNS Failover (5 minutes)

1. Update Route 53 weights to redirect traffic to DR region:
   ```bash
   aws route53 change-resource-record-sets \
     --hosted-zone-id $ZONE_ID \
     --change-batch file://failover-dns-change.json
   ```
2. Verify DNS propagation:
   ```bash
   dig api.bib.example.com
   ```
3. Monitor incoming traffic shifting to DR region

### Phase 3: Database Promotion (10 minutes)

Promote databases in tier order (Tier 1 first):

1. **Tier 1 - Ledger and Payment databases:**
   ```bash
   aws rds promote-read-replica --db-instance-identifier bib-prod-ledger-replica
   aws rds promote-read-replica --db-instance-identifier bib-prod-payment-replica
   ```
2. **Tier 2 - Account, Lending, Deposit, Card databases:**
   ```bash
   aws rds promote-read-replica --db-instance-identifier bib-prod-account-replica
   aws rds promote-read-replica --db-instance-identifier bib-prod-lending-replica
   aws rds promote-read-replica --db-instance-identifier bib-prod-deposit-replica
   aws rds promote-read-replica --db-instance-identifier bib-prod-card-replica
   ```
3. **Tier 3 - Identity, Fraud, FX, Reporting databases:**
   ```bash
   aws rds promote-read-replica --db-instance-identifier bib-prod-identity-replica
   aws rds promote-read-replica --db-instance-identifier bib-prod-fraud-replica
   aws rds promote-read-replica --db-instance-identifier bib-prod-fx-replica
   aws rds promote-read-replica --db-instance-identifier bib-prod-reporting-replica
   ```
4. Wait for each promotion to complete (check status):
   ```bash
   aws rds describe-db-instances --db-instance-identifier bib-prod-ledger-replica \
     --query 'DBInstances[0].DBInstanceStatus'
   ```

### Phase 4: Kafka Failover (10 minutes)

1. Verify MirrorMaker has replicated all critical topics
2. Update Kafka consumer configurations to point to DR cluster
3. Restart service deployments to pick up new Kafka endpoints:
   ```bash
   kubectl --context dr rollout restart deployment -n bib
   ```

### Phase 5: Service Verification (15 minutes)

1. Run health checks on all services:
   ```bash
   for svc in ledger account payment fx deposit identity lending fraud card reporting; do
     curl -s https://api-dr.bib.example.com/api/v1/${svc}/healthz
   done
   ```
2. Run smoke tests:
   ```bash
   cd e2e && go test -tags=dr -run TestDRSmoke ./...
   ```
3. Verify ledger balances are consistent
4. Check for any stuck or in-flight transactions

### Phase 6: Communication

1. Update status page
2. Notify operations team via PagerDuty
3. Send customer notification if customer-facing impact exceeds 30 minutes
4. Log the failover event with timestamps in the incident tracker

## Failback Procedure

After the primary region is restored:

1. **Do NOT immediately fail back** - verify primary region stability for at
   least 1 hour
2. Set up replication from DR back to primary
3. Wait for replication to catch up (zero lag)
4. Schedule a maintenance window for failback
5. Follow the failover procedure in reverse, promoting primary databases
6. Update DNS to shift traffic back to primary region
7. Re-establish DR replication from primary to DR

## Contacts

| Role | Contact |
|------|---------|
| DR Lead | On-call SRE (PagerDuty) |
| Database Admin | DBA on-call |
| Network Engineer | NetOps on-call |
| Executive Sponsor | VP Engineering |

## Post-Incident

After every failover (planned or unplanned):

1. Conduct a post-incident review within 48 hours
2. Update this runbook with lessons learned
3. File tickets for any automation improvements
4. Re-test DR readiness within 2 weeks
