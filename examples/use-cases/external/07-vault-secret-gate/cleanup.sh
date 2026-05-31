#!/usr/bin/env bash
set -e
kubectl delete webapp my-app my-app-expired --ignore-not-found
kubectl delete -f ../crd.yaml --ignore-not-found
