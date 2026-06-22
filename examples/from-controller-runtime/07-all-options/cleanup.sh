#!/usr/bin/env bash
kubectl delete -f crs/ --ignore-not-found
kubectl delete -f crds/ --ignore-not-found
