#!/usr/bin/env bash
set -euo pipefail
kubectl delete workerpool my-pool --ignore-not-found
kubectl delete crd workerpools.demo.orkestra.io --ignore-not-found
