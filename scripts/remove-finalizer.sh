#!/usr/bin/env bash
set -euo pipefail

FINALIZER="orkestra.orkspace.io/finalizer"

# All kinds from your watchers (including commented ones)
KINDS=(
  Pod
  Service
  Namespace
  ServiceAccount
  PersistentVolume
  PersistentVolumeClaim
  Deployment
  StatefulSet
  DaemonSet
  ReplicaSet
  Job
  CronJob
  ConfigMap
  ServiceAccount
  Secret
  Ingress
  Event
)

echo "=== Orkestra Finalizer Cleanup ==="
echo "Target finalizer: $FINALIZER"
echo

# Loop through all namespaces
for ns in $(kubectl get ns -o jsonpath='{.items[*].metadata.name}'); do
  echo "📁 Namespace: $ns"

  # Loop through all resource kinds
  for kind in "${KINDS[@]}"; do
    # Skip kinds that are cluster-scoped when namespace != ""
    if [[ "$kind" == "PersistentVolume" ]]; then
      # PVs are cluster-scoped → ignore namespace
      names=$(kubectl get "$kind" --ignore-not-found -o jsonpath='{.items[*].metadata.name}')
    else
      names=$(kubectl get "$kind" -n "$ns" --ignore-not-found -o jsonpath='{.items[*].metadata.name}')
    fi

    [[ -z "$names" ]] && continue

    echo "  🔎 Checking $kind objects…"

    for name in $names; do
      # Fetch JSON and check if finalizer exists
      if [[ "$kind" == "PersistentVolume" ]]; then
        has_finalizer=$(kubectl get "$kind" "$name" -o json | jq -r \
          --arg f "$FINALIZER" '.metadata.finalizers // [] | index($f)')
      else
        has_finalizer=$(kubectl get "$kind" "$name" -n "$ns" -o json | jq -r \
          --arg f "$FINALIZER" '.metadata.finalizers // [] | index($f)')
      fi

      if [[ "$has_finalizer" != "null" ]]; then
        echo "      ⚠️  Removing finalizer from $kind/$name"

        if [[ "$kind" == "PersistentVolume" ]]; then
          kubectl patch "$kind" "$name" \
            --type=json \
            -p="[{'op': 'remove', 'path': '/metadata/finalizers'}]"
        else
          kubectl patch "$kind" "$name" -n "$ns" \
            --type=json \
            -p="[{'op': 'remove', 'path': '/metadata/finalizers'}]"
        fi
      fi
    done
  done

  echo
done

echo "🎉 Cleanup complete."
