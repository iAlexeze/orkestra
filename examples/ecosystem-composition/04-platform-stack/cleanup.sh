#!/usr/bin/env bash
kubectl delete -f cr-app.yaml --ignore-not-found
kubectl delete -f cr-security.yaml --ignore-not-found
kubectl delete -f cr-infra.yaml --ignore-not-found
helm uninstall cert-manager -n cert-manager
helm uninstall kube-prometheus-stack -n monitoring
helm uninstall crossplane -n crossplane-system
kubectl delete -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl delete namespace argocd --ignore-not-found
