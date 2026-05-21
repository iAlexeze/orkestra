#!/bin/bash
# Cleanup script for 03-conditional-children.
# Deletes sample CRs first, then the CRDs.

kubectl delete -f cr-dev.yaml --ignore-not-found
kubectl delete -f cr-prod.yaml --ignore-not-found
kubectl delete -f crd-appenvironment.yaml --ignore-not-found
kubectl delete -f crd-cachecluster.yaml --ignore-not-found
kubectl delete -f crd-searchindex.yaml --ignore-not-found

echo "Done. Stop 'ork run' with Ctrl+C if still running."
