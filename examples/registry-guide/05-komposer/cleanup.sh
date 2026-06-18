#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 05-komposer example..."

kubectl delete -f ../02-katalog-api/cr.yaml --ignore-not-found
kubectl delete -f ../03-katalog-cache/cr.yaml --ignore-not-found
kubectl delete -f ../04-katalog-platform/cr.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system --ignore-not-found
kubectl delete -f bundle.yaml --ignore-not-found
kubectl delete -f ../02-katalog-api/crd.yaml --ignore-not-found
kubectl delete -f ../03-katalog-cache/crd.yaml --ignore-not-found
kubectl delete -f ../04-katalog-platform/crd.yaml --ignore-not-found

echo "✓ Done."
