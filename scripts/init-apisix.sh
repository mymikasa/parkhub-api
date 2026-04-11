#!/usr/bin/env bash
set -euo pipefail

APISIX_ADMIN="http://apisix:9180/apisix/admin"
API_KEY="FbhKIQyehHlnrBtFLsTBzqIAfechxxoQ"

wait_for_apisix() {
  local retries=30
  local interval=2
  for i in $(seq 1 "$retries"); do
    if curl -sf -o /dev/null "${APISIX_ADMIN}/routes" -H "X-API-KEY: ${API_KEY}"; then
      echo "APISIX Admin API is ready"
      return 0
    fi
    echo "Waiting for APISIX Admin API... ($i/$retries)"
    sleep "$interval"
  done
  echo "ERROR: APISIX Admin API not ready after $((retries * interval)) seconds"
  return 1
}

wait_for_apisix

echo "Creating upstream: monolith-grpc"
curl -sf "${APISIX_ADMIN}/upstreams/1" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "monolith-grpc",
    "type": "roundrobin",
    "scheme": "grpc",
    "nodes": {
      "monolith:50051": 1
    },
    "checks": {
      "active": {
        "type": "http",
        "http_path": "/healthz",
        "healthy": {
          "interval": 5,
          "successes": 2
        },
        "unhealthy": {
          "interval": 5,
          "http_failures": 3
        }
      }
    }
  }' && echo ""

echo "Creating route: grpc-proxy"
curl -sf "${APISIX_ADMIN}/routes/1" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "grpc-proxy",
    "uri": "/*",
    "upstream_id": "1",
    "plugins": {
      "prometheus": {}
    }
  }' && echo ""

echo "APISIX configuration complete"
