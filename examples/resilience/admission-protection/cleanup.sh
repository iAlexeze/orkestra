#!/usr/bin/env bash
set -euo pipefail
kubectl delete -f cr-bad.yaml --ignore-not-found
kubectl delete -f cr-good.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
