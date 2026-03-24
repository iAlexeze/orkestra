#!/bin/bash
# tests/e2e/activation/test.sh
set -e

echo "Testing: Missing CRD appears after startup"

# Start Orkestra WITHOUT installing CRD
/tmp/ork run --katalog tests/fixtures/katalogs/missing-crd.yaml &
ORK_PID=$!
sleep 5

# Verify health shows degraded
curl -s localhost:8080/katalog/missing-crd/health | jq -e '.healthy == false'
HEALTH_DEGRADED=$?

# Install CRD
kubectl apply -f tests/fixtures/crds/missing-crd.yaml
sleep 10

# Verify health becomes healthy
curl -s localhost:8080/katalog/missing-crd/health | jq -e '.healthy == true'
HEALTH_HEALTHY=$?

# Cleanup
kill $ORK_PID
kubectl delete -f tests/fixtures/crds/missing-crd.yaml

if [ $HEALTH_DEGRADED -eq 0 ] && [ $HEALTH_HEALTHY -eq 0 ]; then
    echo "✅ Activation E2E test passed"
    exit 0
else
    echo "❌ Activation E2E test failed"
    exit 1
fi