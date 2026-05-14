#!/bin/bash
# Cleanup script for 02-status-propagation.
# Deletes the sample CR first, then the CRDs.

kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f crd-datapipeline.yaml --ignore-not-found
kubectl delete -f crd-connector.yaml --ignore-not-found

echo "Done. Stop 'ork run' with Ctrl+C if still running."
