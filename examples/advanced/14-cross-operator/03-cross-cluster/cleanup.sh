#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 13-cross-operator / 03-cross-cluster..."

echo "→ cluster-a (producer)"
kubectl config use-context kind-orkestra-a
kubectl delete -f cr-producer.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "→ cluster-b (consumer)"
kubectl config use-context kind-orkestra-b
kubectl delete -f cr-consumer.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

echo "→ deleting Kind clusters"
kind delete cluster --name orkestra-a 2>/dev/null || true
kind delete cluster --name orkestra-b 2>/dev/null || true

echo "✓ Done."
