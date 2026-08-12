#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up conditional-reconciliation..."
kubectl delete app my-app --ignore-not-found
kubectl delete route my-route --ignore-not-found
kubectl delete -f crd-app.yaml --ignore-not-found
kubectl delete -f crd-route.yaml --ignore-not-found
echo "Done. Stop 'ork run' with Ctrl+C if still running."
