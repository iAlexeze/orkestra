#!/usr/bin/env bash
kubectl delete -f cr.yaml --ignore-not-found
kubectl delete -f crd.yaml --ignore-not-found
helm uninstall webapp-operator --namespace webapp-system --ignore-not-found
kubectl delete namespace webapp-system --ignore-not-found
