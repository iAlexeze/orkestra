#!/usr/bin/env bash
set -euo pipefail

echo "Cleaning up admission example..."

kubectl delete -f cr-valid.yaml   --ignore-not-found
kubectl delete -f cr-bad.yaml     --ignore-not-found
kubectl delete -f cr-mutated.yaml --ignore-not-found
kubectl delete -f crd.yaml        --ignore-not-found
kubectl delete -f bundle.yaml     --ignore-not-found

echo "Done. Stop 'helm uninstall orkestra -n orkestra-system' if Orkestra is still running."
