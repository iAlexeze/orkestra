#!/usr/bin/env bash
set -euo pipefail
kubectl delete workerservice my-worker --ignore-not-found
kubectl delete crd workerservices.demo.orkestra.io --ignore-not-found
