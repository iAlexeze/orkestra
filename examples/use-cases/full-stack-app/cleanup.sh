#!/usr/bin/env bash
set -euo pipefail

echo "Cleaning up use-cases..."

kubectl delete -f 01-multi-region/crd.yaml --ignore-not-found
kubectl delete -f 02-external-gate/crd.yaml --ignore-not-found
kubectl delete -f 03-cross-crd/crd.yaml --ignore-not-found
kubectl delete -f 04-once-secret/crd.yaml --ignore-not-found
kubectl delete -f 05-anyof/crd.yaml --ignore-not-found
kubectl delete -f 06-full-stack/crd.yaml --ignore-not-found

echo "✓ Done. Stop 'ork run' with Ctrl+C if still running."
