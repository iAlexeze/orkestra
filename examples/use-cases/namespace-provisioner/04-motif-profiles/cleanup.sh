#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up namespace-provisioner/04-motif-profiles..."
kubectl delete -f ./cr.yaml --ignore-not-found
kubectl delete -f ../crd.yaml --ignore-not-found
kubectl delete -f ../setup.yaml --ignore-not-found
echo "✓ Done."
