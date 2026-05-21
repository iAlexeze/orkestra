#!/bin/bash
# Cleanup script for 05-forEach-sharding.
# Deletes the sample CR first (cascades to Shard children), then the CRDs.

kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f crd-shardedstore.yaml --ignore-not-found
kubectl delete -f crd-shard.yaml --ignore-not-found

echo "Done. Stop 'ork run' with Ctrl+C if still running."
