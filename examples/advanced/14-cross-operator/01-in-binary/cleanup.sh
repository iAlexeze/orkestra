#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 13-cross-operator / 01-in-binary..."

kubectl delete -f cr-consumer.yaml  --ignore-not-found
kubectl delete -f cr-producer.yaml  --ignore-not-found
kubectl delete -f crd.yaml          --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
