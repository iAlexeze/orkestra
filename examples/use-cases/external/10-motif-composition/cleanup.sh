#!/usr/bin/env bash
set -e
kubectl delete webapp my-app --ignore-not-found
kubectl delete worker my-worker --ignore-not-found
kubectl delete -f crd-webapp.yaml --ignore-not-found
kubectl delete -f crd-worker.yaml --ignore-not-found
