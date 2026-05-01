#!/bin/bash

# Usage: ./delete_problem_pods.sh [NAMESPACE]
# If NAMESPACE is not provided, uses 'default'

NAMESPACE="${1:-default}"

# Statuses to target
STATUSES=(
    "ImagePullBackOff"
    "Completed"
    "ContainerStatusUnknown"
    "CrashLoopBackOff"
    "Running"
    "ErrImagePull"
    "ContainerCreating"
    "ErrImagePull"
)

# Build kubectl get pods command with status filtering
# We'll get all pods then filter by status using grep/awk because kubectl's json output is more reliable
echo "Fetching pods in namespace: $NAMESPACE"

# Get all pods with their status
pods=$(kubectl get pods -n "$NAMESPACE" --no-headers)

# Process each pod
while IFS= read -r line; do
    # Extract pod name and status
    pod_name=$(echo "$line" | awk '{print $1}')
    pod_status=$(echo "$line" | awk '{print $3}')

    # Check if status matches any of the target statuses
    match=0
    for status in "${STATUSES[@]}"; do
        if [[ "$pod_status" == "$status" ]]; then
            match=1
            break
        fi
    done

    if [[ $match -eq 1 ]]; then
        echo "Processing pod: $pod_name (status: $pod_status)"

        # Patch finalizers to empty array
        echo "  Patching finalizers to []..."
        if kubectl patch pod "$pod_name" -n "$NAMESPACE" --type='merge' \
            --patch='{"metadata":{"finalizers":[]}}' >/dev/null 2>&1; then
            echo "  Finalizers cleared successfully."
        else
            echo "  No finalizers to clear or patch failed (continuing with deletion)."
        fi

        # Delete the pod
        echo "  Deleting pod $pod_name..."
        if kubectl delete pod "$pod_name" -n "$NAMESPACE"; then
            echo "  Pod $pod_name deleted."
        else
            echo "  ERROR: Failed to delete pod $pod_name" >&2
        fi
        echo "---"
    fi
done <<< "$pods"

echo "Done."