#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up conditional-resources..."
kubectl delete platform my-platform --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
echo "Done. Stop 'ork run' with Ctrl+C if still running."
