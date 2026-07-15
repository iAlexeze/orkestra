#!/usr/bin/env bash
set -euo pipefail
kubectl delete apiserver my-api --ignore-not-found
kubectl delete crd apiservers.demo.orkestra.io --ignore-not-found
