#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up profiles/05-pdb..."
kubectl delete -f ../cr.yaml --ignore-not-found
kubectl delete -f ../crd.yaml --ignore-not-found
echo "✓ Done."
