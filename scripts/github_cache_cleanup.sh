#!/usr/bin/env bash
set -euo pipefail

OWNER="orkspace"
REPO="orkestra"
OLDER_THAN_DAYS=2

now=$(date +%s)

gh api "/repos/$OWNER/$REPO/actions/caches" --paginate --jq '.actions_caches[]' | while read -r cache; do
  id=$(echo "$cache" | jq -r '.id')
  key=$(echo "$cache" | jq -r '.key')
  created_at=$(echo "$cache" | jq -r '.created_at')
  created_ts=$(date -d "$created_at" +%s)
  age_days=$(( (now - created_ts) / 86400 ))
  if [ "$age_days" -ge "$OLDER_THAN_DAYS" ]; then
    echo "Deleting cache id=$id key=$key age=${age_days}d"
    gh api -X DELETE "/repos/$OWNER/$REPO/actions/caches/$id" --silent
  fi
done