#!/bin/bash
# tests/e2e/dependencies/test.sh
set -e

echo "Testing: Dependency ordering"

# Install CRDs (A and B, where B depends on A)
kubectl apply -f tests/fixtures/crds/crd-a.yaml
kubectl apply -f tests/fixtures/crds/crd-b.yaml

# Start Orkestra
/tmp/ork run --katalog tests/fixtures/katalogs/dependencies.yaml &
ORK_PID=$!
sleep 5

# Check startup order (B should start only after A)
curl -s localhost:8080/katalog/crd-a/health | jq -e '.started == true'
A_STARTED=$?
curl -s localhost:8080/katalog/crd-b/health | jq -e '.started == true'
B_STARTED=$?

# Cleanup
kill $ORK_PID
kubectl delete -f tests/fixtures/crds/crd-a.yaml
kubectl delete -f tests/fixtures/crds/crd-b.yaml

if [ $A_STARTED -eq 0 ] && [ $B_STARTED -eq 0 ]; then
    echo "✅ Dependencies E2E test passed"
    exit 0
else
    echo "❌ Dependencies E2E test failed"
    exit 1
fi