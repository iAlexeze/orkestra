#!/usr/bin/env bash
set -euo pipefail

echo "Cleaning up namespace-protection example..."

kubectl delete -f cr-allowed.yaml  --ignore-not-found
kubectl delete -f cr-blocked.yaml  --ignore-not-found
kubectl delete namespace production --ignore-not-found
kubectl delete namespace staging    --ignore-not-found
kubectl delete -f crd.yaml          --ignore-not-found
kubectl delete -f bundle.yaml       --ignore-not-found

echo "Done. Stop 'helm uninstall orkestra -n orkestra-system' if Orkestra is still running."
