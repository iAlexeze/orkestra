#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up implementation-evolution example..."

kubectl delete -f 02-api-team-upgrades/cr.yaml --ignore-not-found
kubectl delete -f ../../04-katalog-platform/cr.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system --ignore-not-found
kubectl delete -f bundle.yaml --ignore-not-found
kubectl delete -f 02-api-team-upgrades/crd.yaml --ignore-not-found
kubectl delete -f ../../04-katalog-platform/crd.yaml --ignore-not-found

echo "✓ Done."
