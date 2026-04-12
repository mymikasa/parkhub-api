#!/usr/bin/env bash
set -euo pipefail

APISIX_ADMIN="http://apisix:9180/apisix/admin"
API_KEY="FbhKIQyehHlnrBtFLsTBzqIAfechxxoQ"
DESC_FILE="/proto/proto-descriptor.pb"

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

# ── Upstream: monolith gRPC ──────────────────────────────────────────
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
        "port": 8080,
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

# ── Proto: FileDescriptorSet (buf build output) ──────────────────────
echo "Registering proto descriptor"
B64=$(base64 -w 0 "$DESC_FILE")
curl -sf "${APISIX_ADMIN}/protos/1" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d "{\"content\": \"${B64}\"}" && echo ""

# ── Routes: TenantService ────────────────────────────────────────────

echo "Creating route: CreateTenant (POST /api/v1/tenants)"
curl -sf "${APISIX_ADMIN}/routes/10" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "tenant-create",
    "methods": ["POST"],
    "uri": "/api/v1/tenants",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "CreateTenant",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: ListTenants (GET /api/v1/tenants)"
curl -sf "${APISIX_ADMIN}/routes/11" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "tenant-list",
    "methods": ["GET"],
    "uri": "/api/v1/tenants",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "ListTenants",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: GetTenant (GET /api/v1/tenants/:id)"
curl -sf "${APISIX_ADMIN}/routes/12" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "tenant-get",
    "methods": ["GET"],
    "uri": "/api/v1/tenants/:tenant_id",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "GetTenant",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: UpdateTenant (PUT /api/v1/tenants/:id)"
curl -sf "${APISIX_ADMIN}/routes/13" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "tenant-update",
    "methods": ["PUT"],
    "uri": "/api/v1/tenants/:tenant_id",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "UpdateTenant",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: DeleteTenant (DELETE /api/v1/tenants/:id)"
curl -sf "${APISIX_ADMIN}/routes/14" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "tenant-delete",
    "methods": ["DELETE"],
    "uri": "/api/v1/tenants/:tenant_id",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "DeleteTenant",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "prometheus": {}
    }
  }' && echo ""

echo "APISIX configuration complete"
