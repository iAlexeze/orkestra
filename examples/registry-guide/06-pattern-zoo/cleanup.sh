#!/usr/bin/env bash
set -euo pipefail
echo "Cleaning up 06-pattern-zoo example..."

kubectl delete -f cr-postgres.yaml -f cr-mysql.yaml -f cr-mongodb.yaml \
  -f cr-redis.yaml -f cr-kafka.yaml -f cr-rabbitmq.yaml \
  -f cr-deployment-stack.yaml \
  -f ../02-katalog-api/cr.yaml -f ../03-katalog-cache/cr.yaml --ignore-not-found
helm uninstall orkestra -n orkestra-system --ignore-not-found
kubectl delete -f bundle.yaml --ignore-not-found
kubectl delete -f ./crds/ --ignore-not-found
kubectl delete -f ../02-katalog-api/crd.yaml --ignore-not-found
kubectl delete -f ../03-katalog-cache/crd.yaml --ignore-not-found

echo "✓ Done."
