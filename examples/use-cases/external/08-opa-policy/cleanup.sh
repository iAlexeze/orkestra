#!/usr/bin/env bash
set -e
kubectl delete webapp my-app deny-my-app --ignore-not-found
kubectl delete -f ../crd.yaml --ignore-not-found
