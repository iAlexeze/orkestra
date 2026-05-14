#!/bin/bash
# Cleanup script for 06-full-platform-composition.
# Deletes sample CRs first (cascading to all child CRs), then the CRDs.

kubectl delete -f cr-small.yaml --ignore-not-found
kubectl delete -f cr-large.yaml --ignore-not-found
kubectl delete -f crd-platform.yaml --ignore-not-found
kubectl delete -f crd-messagequeue.yaml --ignore-not-found
kubectl delete -f crd-objectstore.yaml --ignore-not-found
kubectl delete -f crd-searchcluster.yaml --ignore-not-found

echo "Done. Stop 'ork run' with Ctrl+C if still running."
