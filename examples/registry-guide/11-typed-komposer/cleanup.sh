#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 11-typed-komposer example..."

kubectl delete -f cr-webapp.yaml -f cr-cache.yaml -f cr-database.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system --ignore-not-found
kubectl delete -f bundle.yaml --ignore-not-found
kubectl delete -f ../02-katalog-api/crd.yaml --ignore-not-found
kubectl delete -f ../03-katalog-cache/crd.yaml --ignore-not-found
kubectl delete -f ../10-hooks-katalog/crd.yaml --ignore-not-found

echo "✓ Done."
