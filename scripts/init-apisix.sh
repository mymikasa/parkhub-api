#!/usr/bin/env bash
set -euo pipefail

APISIX_ADMIN="http://apisix:9180/apisix/admin"
API_KEY="FbhKIQyehHlnrBtFLsTBzqIAfechxxoQ"
DESC_FILE="/proto/proto-descriptor.pb"

JWT_INJECT='return function(conf, ctx) local pl = ctx.jwt_auth_payload; if pl then ngx.req.set_header(\"x-user-id\", pl.user_id or \"\"); ngx.req.set_header(\"x-tenant-id\", pl.tenant_id or \"\"); ngx.req.set_header(\"x-user-role\", pl.role or \"\") end end'

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

# ── Consumer: jwt-auth (RS256) ───────────────────────────────────────
echo "Creating consumer: jwt-auth"
PUB_KEY_FILE="/keys/jwt_public.pem"
if [ -f "$PUB_KEY_FILE" ]; then
  PUB_KEY=$(awk 'NF {sub(/\r/,""); printf "%s\\n",$0}' "$PUB_KEY_FILE")
  echo "Creating consumer with public key from $PUB_KEY_FILE"
  curl -sf "${APISIX_ADMIN}/consumers/jwt" \
    -H "X-API-KEY: ${API_KEY}" \
    -X PUT \
    -d "{
      \"username\": \"jwt\",
      \"plugins\": {
        \"jwt-auth\": {
          \"algorithm\": \"RS256\",
          \"key\": \"parkhub-2026-04\",
          \"public_key\": \"${PUB_KEY}\"
        }
      }
    }" && echo ""
else
  echo "WARNING: No public key found at $PUB_KEY_FILE, creating consumer without public key"
  curl -sf "${APISIX_ADMIN}/consumers/jwt" \
    -H "X-API-KEY: ${API_KEY}" \
    -X PUT \
    -d '{
      "username": "jwt",
      "plugins": {
        "jwt-auth": {
          "algorithm": "RS256",
          "key": "parkhub-2026-04"
        }
      }
    }' && echo ""
fi

# ── Plugin Metadata: opentelemetry ───────────────────────────────────
echo "Registering opentelemetry plugin metadata"
curl -sf "${APISIX_ADMIN}/plugin_metadata/opentelemetry" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "collector": {
      "address": "tempo:4318",
      "request_timeout": 3
    },
    "resource": {
      "service.name": "APISIX"
    },
    "batch_span_processor": {
      "max_queue_size": 1024,
      "batch_timeout": 2,
      "inactive_timeout": 1,
      "max_export_batch_size": 16
    }
  }' && echo ""

# ══════════════════════════════════════════════════════════════════════
# Routes: TenantService (protected with jwt-auth)
# ══════════════════════════════════════════════════════════════════════

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
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": ["'"${JWT_INJECT}"'"]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "CreateTenant",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
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
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": ["'"${JWT_INJECT}"'"]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "ListTenants",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
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
    "uri": "/api/v1/tenants/*",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/tenants/(.+)$\"); if id then ngx.req.set_uri_args({tenant_id = id}) end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "GetTenant",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
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
    "uri": "/api/v1/tenants/*",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/tenants/(.+)$\"); ngx.req.read_body(); local body = ngx.req.get_body_data(); if body then local cjson = require(\"cjson.safe\"); local t = cjson.decode(body); if t and id then t.tenant_id = id; ngx.req.set_body_data(cjson.encode(t)) end end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "UpdateTenant",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
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
    "uri": "/api/v1/tenants/*",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/tenants/(.+)$\"); if id then ngx.req.set_uri_args({tenant_id = id}) end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.TenantService",
        "method": "DeleteTenant",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

# ══════════════════════════════════════════════════════════════════════
# Routes: UserService (protected with jwt-auth)
# ══════════════════════════════════════════════════════════════════════

echo "Creating route: CreateUser (POST /api/v1/users)"
curl -sf "${APISIX_ADMIN}/routes/20" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-create",
    "methods": ["POST"],
    "uri": "/api/v1/users",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": ["'"${JWT_INJECT}"'"]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "CreateUser",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: ListUsers (GET /api/v1/users)"
curl -sf "${APISIX_ADMIN}/routes/21" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-list",
    "methods": ["GET"],
    "uri": "/api/v1/users",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": ["'"${JWT_INJECT}"'"]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "ListUsers",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: GetUser (GET /api/v1/users/:id)"
curl -sf "${APISIX_ADMIN}/routes/22" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-get",
    "methods": ["GET"],
    "uri": "/api/v1/users/*",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/users/(.+)$\"); if id then ngx.req.set_uri_args({user_id = id}) end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "GetUser",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: UpdateUser (PUT /api/v1/users/:id)"
curl -sf "${APISIX_ADMIN}/routes/23" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-update",
    "methods": ["PUT"],
    "uri": "/api/v1/users/*",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/users/(.+)$\"); ngx.req.read_body(); local body = ngx.req.get_body_data(); if body then local cjson = require(\"cjson.safe\"); local t = cjson.decode(body); if t and id then t.user_id = id; ngx.req.set_body_data(cjson.encode(t)) end end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "UpdateUser",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: FreezeUser (PUT /api/v1/users/:id/freeze)"
curl -sf "${APISIX_ADMIN}/routes/24" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-freeze",
    "methods": ["PUT"],
    "uri": "/api/v1/users/*/freeze",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/users/(.+)/freeze$\"); if id then ngx.req.read_body(); local cjson = require(\"cjson.safe\"); local body = ngx.req.get_body_data() or \"{}\"; local t = cjson.decode(body) or {}; t.user_id = id; ngx.req.set_body_data(cjson.encode(t)) end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "FreezeUser",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: UnfreezeUser (PUT /api/v1/users/:id/unfreeze)"
curl -sf "${APISIX_ADMIN}/routes/25" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-unfreeze",
    "methods": ["PUT"],
    "uri": "/api/v1/users/*/unfreeze",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/users/(.+)/unfreeze$\"); if id then ngx.req.read_body(); local cjson = require(\"cjson.safe\"); local body = ngx.req.get_body_data() or \"{}\"; local t = cjson.decode(body) or {}; t.user_id = id; ngx.req.set_body_data(cjson.encode(t)) end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "UnfreezeUser",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: ResetPassword (PUT /api/v1/users/:id/reset-password)"
curl -sf "${APISIX_ADMIN}/routes/26" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-reset-password",
    "methods": ["PUT"],
    "uri": "/api/v1/users/*/reset-password",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/users/(.+)/reset%-password$\"); ngx.req.read_body(); local body = ngx.req.get_body_data(); if body then local cjson = require(\"cjson.safe\"); local t = cjson.decode(body); if t and id then t.user_id = id; ngx.req.set_body_data(cjson.encode(t)) end end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "ResetPassword",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: UpdateProfile (PUT /api/v1/users/:id/profile)"
curl -sf "${APISIX_ADMIN}/routes/27" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-update-profile",
    "methods": ["PUT"],
    "uri": "/api/v1/users/*/profile",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/users/(.+)/profile$\"); ngx.req.read_body(); local body = ngx.req.get_body_data(); if body then local cjson = require(\"cjson.safe\"); local t = cjson.decode(body); if t and id then t.user_id = id; ngx.req.set_body_data(cjson.encode(t)) end end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "UpdateProfile",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: ChangePassword (PUT /api/v1/users/:id/change-password)"
curl -sf "${APISIX_ADMIN}/routes/28" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-change-password",
    "methods": ["PUT"],
    "uri": "/api/v1/users/*/change-password",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": [
          "'"${JWT_INJECT}"'",
          "return function(conf, ctx) local id = ngx.var.uri:match(\"^/api/v1/users/(.+)/change%-password$\"); ngx.req.read_body(); local body = ngx.req.get_body_data(); if body then local cjson = require(\"cjson.safe\"); local t = cjson.decode(body); if t and id then t.user_id = id; ngx.req.set_body_data(cjson.encode(t)) end end end"
        ]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "ChangePassword",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: ImportUsers (POST /api/v1/users/import)"
curl -sf "${APISIX_ADMIN}/routes/29" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-import",
    "methods": ["POST"],
    "uri": "/api/v1/users/import",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": ["'"${JWT_INJECT}"'"]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "ImportUsers",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

# ══════════════════════════════════════════════════════════════════════
# Routes: AuthService (public, no jwt-auth)
# ══════════════════════════════════════════════════════════════════════

echo "Creating route: GetCurrentUser (GET /identity/v1/users/me)"
curl -sf "${APISIX_ADMIN}/routes/34" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "user-me",
    "methods": ["GET"],
    "uri": "/identity/v1/users/me",
    "upstream_id": "1",
    "plugins": {
      "jwt-auth": { "store_in_ctx": true },
      "serverless-pre-function": {
        "phase": "access",
        "functions": ["'"${JWT_INJECT}"'"]
      },
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.UserService",
        "method": "GetCurrentUser",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: Login (POST /identity/v1/auth/login)"
curl -sf "${APISIX_ADMIN}/routes/30" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "auth-login",
    "methods": ["POST"],
    "uri": "/identity/v1/auth/login",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.AuthService",
        "method": "Login",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: RefreshToken (POST /identity/v1/auth/refresh)"
curl -sf "${APISIX_ADMIN}/routes/31" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "auth-refresh",
    "methods": ["POST"],
    "uri": "/identity/v1/auth/refresh",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.AuthService",
        "method": "RefreshToken",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: Logout (POST /identity/v1/auth/logout)"
curl -sf "${APISIX_ADMIN}/routes/32" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "auth-logout",
    "methods": ["POST"],
    "uri": "/identity/v1/auth/logout",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.AuthService",
        "method": "Logout",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: GetJWKS (GET /identity/v1/auth/jwks)"
curl -sf "${APISIX_ADMIN}/routes/33" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "auth-jwks",
    "methods": ["GET"],
    "uri": "/identity/v1/auth/jwks",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.AuthService",
        "method": "GetJWKS",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "APISIX configuration complete"

echo "Creating route: SendCode (POST /identity/v1/auth/sms/send)"
curl -sf "${APISIX_ADMIN}/routes/40" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "sms-send-code",
    "methods": ["POST"],
    "uri": "/identity/v1/auth/sms/send",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.sms.v1.SmsService",
        "method": "SendCode",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "Creating route: SmsLogin (POST /identity/v1/auth/sms/login)"
curl -sf "${APISIX_ADMIN}/routes/41" \
  -H "X-API-KEY: ${API_KEY}" \
  -X PUT \
  -d '{
    "name": "auth-sms-login",
    "methods": ["POST"],
    "uri": "/identity/v1/auth/sms/login",
    "upstream_id": "1",
    "plugins": {
      "grpc-transcode": {
        "proto_id": "1",
        "service": "parkhub.identity.v1.AuthService",
        "method": "SmsLogin",
        "pb_option": ["enum_as_name", "int64_as_number"]
      },
      "opentelemetry": { "sampler": { "name": "always_on" } },
      "prometheus": {}
    }
  }' && echo ""

echo "APISIX SMS routes configuration complete"
