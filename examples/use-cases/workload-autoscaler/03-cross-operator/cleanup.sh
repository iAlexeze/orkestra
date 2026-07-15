#!/usr/bin/env bash
set -euo pipefail
kubectl delete jobworker my-worker --ignore-not-found
kubectl delete jobqueue my-queue --ignore-not-found
kubectl delete crd jobworkers.demo.orkestra.io --ignore-not-found
kubectl delete crd jobqueues.demo.orkestra.io --ignore-not-found
