#!/usr/bin/env bash
set -euo pipefail

echo "Cleaning up deletion-protection example..."

kubectl delete -f cr-app.yaml        --ignore-not-found
kubectl delete -f cr-database.yaml   --ignore-not-found
kubectl delete -f cr-cache.yaml      --ignore-not-found
kubectl delete -f cr-unprotected.yaml --ignore-not-found
kubectl delete -f crd-protected.yaml  --ignore-not-found
kubectl delete -f crd-unprotected.yaml --ignore-not-found
kubectl delete -f bundle.yaml         --ignore-not-found

echo "Done. Stop 'helm uninstall orkestra -n orkestra-system' if Orkestra is still running."
