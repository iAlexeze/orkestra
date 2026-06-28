#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up profiles/06-networkpolicy..."
kubectl delete -f ../cr.yaml --ignore-not-found
kubectl delete -f ../crd.yaml --ignore-not-found
echo "✓ Done."
