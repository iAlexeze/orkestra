#!/bin/bash
# Cleanup script for 04-drift-correction.
# Deletes the sample CR first, then the CRDs.

kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f crd-database.yaml --ignore-not-found
kubectl delete -f crd-backuppolicy.yaml --ignore-not-found

echo "Done. Stop 'ork run' with Ctrl+C if still running."
