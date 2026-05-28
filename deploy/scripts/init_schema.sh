#!/usr/bin/env bash
set -euo pipefail

SUBJECT="order-events-value"
URL="${SCHEMA_REGISTRY_URL}/subjects/${SUBJECT}/versions"

echo "Publishing schema to ${URL}..."

SCHEMA=$(cat /schemas/order-event-value.json | tr -d '\n' | sed 's/"/\\"/g')
PAYLOAD="{\"schema\": \"${SCHEMA}\"}"

RESPONSE=$(curl -s -X POST \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d "${PAYLOAD}" \
  "${URL}")

if echo "$RESPONSE" | grep -q "id"; then
  echo "Success!"
  echo "Response: $RESPONSE"
else
  echo "Failed to publish schema!"
  echo "Response: $RESPONSE"
  exit 1
fi
