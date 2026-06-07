#!/usr/bin/env bash
set -e

kubectl delete certificate my-tls-cert --ignore-not-found
kubectl delete clusterissuer selfsigned-issuer --ignore-not-found
kubectl delete secret my-tls-secret --ignore-not-found
helm uninstall cert-manager --namespace cert-manager --ignore-not-found
kubectl delete namespace cert-manager --ignore-not-found
