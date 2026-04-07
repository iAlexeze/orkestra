#!/bin/bash
# tests/e2e/website/test.sh
set -e

echo "Testing: Website CRD → Deployment + Service"

# Install CRD
kubectl apply -f examples/website/website-crd.yaml

# Start Orkestra
/tmp/ork run --katalog examples/website/website-katalog.yaml &
ORK_PID=$!
sleep 5

# Apply CR
kubectl apply -f examples/website/website-cr.yaml
sleep 5

# Verify Deployment
kubectl get deployment test-website -n default
DEPLOYMENT_EXISTS=$?

# Verify Service
kubectl get service test-website-svc -n default
SERVICE_EXISTS=$?

# Verify Health Endpoint
curl -s localhost:8080/katalog/website/health | jq -e '.healthy == true'
HEALTH_OK=$?

# Verify Metrics
curl -s localhost:8080/metrics | grep controller_reconcile_total
METRICS_EXIST=$?

# Cleanup
kill $ORK_PID

if [ $DEPLOYMENT_EXISTS -eq 0 ] && [ $SERVICE_EXISTS -eq 0 ] && [ $HEALTH_OK -eq 0 ] && [ $METRICS_EXIST -eq 0 ]; then
    echo "✅ Website E2E test passed"
    exit 0
else
    echo "❌ Website E2E test failed"
    exit 1
fi