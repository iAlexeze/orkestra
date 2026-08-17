#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up blockchainappwithtargets hooks operator..."
kubectl delete blockchainappwithtargets 03-hooks-targets-my-chain --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
echo "✓ Done. Stop 'ork run' with Ctrl+C if still running locally."
