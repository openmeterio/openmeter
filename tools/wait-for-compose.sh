#!/usr/bin/env bash
set -euo pipefail

# Wait for docker compose services to be healthy (or if no health check is available, just running)
#
# Usage:
#   ./tools/wait-for-compose.sh postgres svix redis
#

# Get the container ID for the service
CONTAINER_IDS=$(docker compose ps -q "$@")
if [[ -z "$CONTAINER_IDS" ]]; then
  echo "Error: Could not find container ids."
  exit 1
fi

checkServices() {
    for CONTAINER_ID in $@; do
        STATUS="$(docker inspect --format='{{.State.Health.Status}}' "$CONTAINER_ID" 2>/dev/null || true)"
        SERVICE_NAME="$(docker inspect --format='{{.Name}}' "$CONTAINER_ID" 2>/dev/null | sed 's|^/||')"
        if [[ "$STATUS" == "healthy" ]]; then
            echo "✅ Service '$SERVICE_NAME' is healthy."
        elif [[ "$STATUS" == "unhealthy" ]]; then
            echo "❌ Service '$SERVICE_NAME' is unhealthy."
            exit 1
        elif [[ "$STATUS" == "" ]]; then
            echo "❓ Service '$SERVICE_NAME' is not providing health checks."
            STATUS="$(docker inspect --format='{{.State.Status}}' "$CONTAINER_ID" 2>/dev/null)"
            if [[ "$STATUS" == "running" ]]; then
                echo "✅ Service '$SERVICE_NAME' is running but not providing health checks."
            elif [[ "$STATUS" == "restarting" ]] || [[ "$STATUS" == "created" ]]; then
                echo "⏳ Service '$SERVICE_NAME' is not running (status: $STATUS)."
                return 1
            else
                echo "❌ Service '$SERVICE_NAME' is not running (status: $STATUS)."
                exit 1
            fi
        else
            echo "⏳ '$SERVICE_NAME' status: $STATUS"
            return 1
        fi
    done
    return 0
}

# Wait until healthy (max 60 attempts, 2s each)
attempts=60
while (( attempts > 0 )); do
  if checkServices "${CONTAINER_IDS[@]}"; then
    exit 0
  fi
  sleep 2
  ((attempts--))
done

echo "❌ Failed to start services: timeout"
exit 1
