#!/usr/bin/env bash
set -e

kubectl delete appcert my-app --ignore-not-found
kubectl delete clusterissuer my-app-issuer --ignore-not-found
kubectl delete certificate my-app-cert -n default --ignore-not-found
kubectl delete secret my-app-tls --ignore-not-found
kubectl delete crd appcerts.demo.orkestra.io --ignore-not-found
helm uninstall cert-manager --namespace cert-manager --ignore-not-found
kubectl delete namespace cert-manager --ignore-not-found
