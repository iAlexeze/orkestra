#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 12-dependencies / 03-cross-cluster..."

echo "→ cluster-a (database)"
kubectl config use-context kind-orkestra-a
kubectl delete -f cr-database.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "→ cluster-b (app)"
kubectl config use-context kind-orkestra-b
kubectl delete -f cr-app.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "→ deleting Kind clusters"
kind delete cluster --name orkestra-a 2>/dev/null || true
kind delete cluster --name orkestra-b 2>/dev/null || true

echo "✓ Done."
