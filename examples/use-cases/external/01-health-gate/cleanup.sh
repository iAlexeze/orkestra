#!/usr/bin/env bash
set -e
kubectl delete webapp my-app-healthy my-app-degraded --ignore-not-found
kubectl delete -f ../crd.yaml --ignore-not-found
