#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up webhook-intake example..."

kubectl delete servicerequests.demo.orkestra.io --all --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found

kubectl delete secret ork-payments-github-secret --ignore-not-found
kubectl delete secret ork-payments-github-app-token --ignore-not-found
kubectl delete secret ork-payments-gitlab-secret --ignore-not-found
kubectl delete secret ork-payments-gitlab-api-token --ignore-not-found
kubectl delete secret ork-slack-signing-secret --ignore-not-found
kubectl delete secret ork-pagerduty-webhook-secret --ignore-not-found

echo "✓ Done."
