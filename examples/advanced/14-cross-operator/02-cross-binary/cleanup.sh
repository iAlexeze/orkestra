#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 13-cross-operator / 02-cross-binary..."

# Consumer namespace
kubectl delete -f cr-consumer.yaml  --ignore-not-found -n consumer-system
# Producer namespace
kubectl delete -f cr-producer.yaml  --ignore-not-found -n producer-system

kubectl delete -f crd.yaml --ignore-not-found

kubectl delete namespace producer-system --ignore-not-found
kubectl delete namespace consumer-system --ignore-not-found

echo "✓ Done."
