#!/bin/bash
# Cleanup script for 01-single-child.
# Deletes the sample CR first (triggering owner-reference cascade),
# then removes the CRDs. Stop 'ork run' with Ctrl+C after running this.

kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f crd-workspace.yaml --ignore-not-found
kubectl delete -f crd-secretvault.yaml --ignore-not-found

echo "Done. Stop 'ork run' with Ctrl+C if still running."
