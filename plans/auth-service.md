# Plan: identity 域 AuthService（JWT 登录 / 刷新 / 登出）

## Context

UserService 的 CRUD、密码哈希（bcrypt）、`GetByUsername` 已就绪，但**完全没有登录 / Token 相关代码**：
- `api/proto/identity/v1/` 没有 auth.proto
- `internal/domains/identity/` 没有 auth service
- `go.mod` 没有 JWT 库
- `internal/middleware/` 没有身份注入 interceptor
- `configs/config.yaml` 没有 JWT 配置

前端经 APISIX 接入后端，APISIX 负责 JWT **验签**，后端负责**签发** Token。本计划在 identity 域新增 AuthService，完成 access/refresh token 的签发、刷新、吊销，并在网关与后端之间打通身份传递链路。

**用户决策**:
- AuthService 与 UserService 平级，放在 `internal/domains/identity/`
- 签名算法用 **RS256**，私钥仅后端持有，公钥提供给网关配置
- 使用 **Access + Refresh Token**
- Refresh Token 存 Redis，支持主动吊销与单次消费

**落地约束**:
- APISIX 当前按静态 `public_key` 配置验签，不把“从后端动态拉 JWKS”当作主路径
- 后端只在“gRPC 入口仅允许网关/内网访问”的前提下信任 APISIX 转发的身份 header
- Refresh Token 轮换必须是**原子消费**，不能用 `Exists -> Revoke` 这种非原子流程

---

## 实现步骤

### 1. Proto 定义

新增 `api/proto/identity/v1/auth.proto`，包 `parkhub.identity.v1`：

```proto
service AuthService {
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
  rpc Logout(LogoutRequest) returns (LogoutResponse);
  rpc GetJWKS(GetJWKSRequest) returns (GetJWKSResponse);
}
```

- `LoginRequest`: `username`, `password`
- `LoginResponse`: `access_token`, `refresh_token`, `access_expires_in`, `token_type = "Bearer"`, `User user`
- `RefreshTokenRequest`: `refresh_token`
- `RefreshTokenResponse`: `access_token`, `refresh_token`, `access_expires_in`, `token_type = "Bearer"`
- `LogoutRequest`: `refresh_token`
- `GetJWKS` 保留为只读公钥导出接口，但**不作为 APISIX 主配置来源**
- 参数校验使用 protovalidate，但不要写“参考 `user.proto`”，而是在 `auth.proto` 里补完整规则

跑 `make proto-gen` 生成代码。

### 2. 依赖与配置

`go.mod` 增加 `github.com/golang-jwt/jwt/v5`

`configs/config.yaml` 新增：

```yaml
auth:
  issuer: parkhub
  access_ttl: 15m
  refresh_ttl: 168h
  private_key_path: configs/keys/jwt_private.pem
  public_key_path: configs/keys/jwt_public.pem
  key_id: parkhub-2026-04
```

`internal/config/config.go`：
- 增加 `AuthConfig`
- 在 `Config` 顶层挂 `Auth AuthConfig`
- 如有需要，补 `AUTH_` 前缀的 env override

`configs/keys/`：
- 增加 `.gitignore`，私钥不入库
- 增加 README，说明如何用 openssl 生成密钥对

### 3. Domain 层

新增 `internal/domains/identity/domain/token.go`：
- `type TokenPair struct { AccessToken, RefreshToken string; AccessExpiresAt, RefreshExpiresAt time.Time }`
- `type Claims struct { TokenUse, UserID, TenantID, Role string; jwt.RegisteredClaims }`

关键要求：
- `TokenUse` 必填，取值至少区分 `access` / `refresh`
- access token 带 `userID/tenantID/role`
- refresh token 只带最少字段：`sub`、`jti`、`token_use=refresh`

新增 `internal/domains/identity/domain/signer.go`：

```go
type TokenSigner interface {
    Sign(claims Claims) (string, error)
    Verify(token string) (*Claims, error)
    JWKS() ([]byte, error)
}
```

新增 `internal/domains/identity/domain/rs256_signer.go`：
- 基于 PEM 私钥/公钥/kid 构造
- `Verify` 统一校验签名、`exp`、`iss`
- `JWKS()` 返回 RFC 7517 JSON

`internal/domains/identity/errs/errors.go` 追加：

```go
ErrInvalidCredentials  = errors.New("invalid username or password")
ErrRefreshTokenInvalid = errors.New("refresh token invalid or expired")
ErrRefreshTokenRevoked = errors.New("refresh token revoked")
```

### 4. Repository（Refresh Token 存储）

参考 `internal/domains/sms/repository/cache/` 的 Redis cache 风格，但接口按 identity 域单独建。

建议新增 `internal/domains/identity/repository/auth_interface.go`：

```go
//go:generate mockgen -source=./auth_interface.go -package=repomocks -destination=./mocks/auth_repo.mock.go RefreshTokenRepo
type RefreshTokenRepo interface {
    Save(ctx context.Context, jti, userID string, ttl time.Duration) error
    Consume(ctx context.Context, jti string) (userID string, ok bool, err error)
    Revoke(ctx context.Context, jti string) error
}
```

不要提供 `Exists()` 作为主流程接口。刷新时要用 `Consume()` 实现单次消费语义：
- 首次 refresh: 原子校验存在并删除，返回 `ok=true`
- 重放 refresh: 返回 `ok=false`

新增 `internal/domains/identity/repository/cache/refresh_token_repo.go`：
- Redis key: `auth:refresh:{jti} -> userID`
- TTL 与 refresh token 一致
- `Consume` 用 Lua 脚本或事务保证原子性

### 5. Service 层

新增 `internal/domains/identity/service/auth_interface.go`，定义 service-layer DTO 与 `AuthService` 接口。

新增 `internal/domains/identity/service/auth_service.go`：
- 依赖：`repository.UserRepo`、`repository.RefreshTokenRepo`、`domain.TokenSigner`、`config.AuthConfig`

`Login`：
1. `userRepo.GetByUsername`
2. 用户不存在或密码错误都返回 `ErrInvalidCredentials`
3. `user.IsActive()` 为 false 返回 `ErrUserFrozen`
4. 生成 access token 和 refresh token，二者 `token_use` 明确区分
5. `refreshRepo.Save(jti, userID, refreshTTL)`
6. 更新 `last_login_at`

`last_login_at` 更新建议：
- 优先同步更新，保证语义简单
- 若后续改异步，不要直接复用请求 `ctx`

`RefreshToken`：
1. `signer.Verify(refresh)` 失败返回 `ErrRefreshTokenInvalid`
2. 校验 `claims.TokenUse == "refresh"`，否则返回 `ErrRefreshTokenInvalid`
3. `refreshRepo.Consume(jti)`，`ok=false` 返回 `ErrRefreshTokenRevoked`
4. 重新查 user，冻结用户拒绝继续签发
5. 生成新的 access + refresh token
6. `refreshRepo.Save(newJTI, userID, refreshTTL)`

`Logout`：
1. `signer.Verify(refresh)`，并校验 `token_use == "refresh"`
2. `refreshRepo.Revoke(jti)`
3. 对已不存在的 jti 允许幂等成功

测试按 TDD 风格补齐：
- 密码错误
- 用户冻结
- refresh 过期
- refresh 被吊销
- refresh 二次使用重放
- access token 误用于 refresh/logout

### 6. gRPC Server

新增 `internal/domains/identity/grpc/auth_server.go`：保持薄壳，只做 proto 和 service DTO 转换。

注意当前仓库风格不是改全局错误表，而是每个 server 自己维护 mapping。参考 `internal/domains/identity/grpc/user_server.go`。

新增 `authErrorMappings`：
- `ErrInvalidCredentials` -> `codes.Unauthenticated`
- `ErrUserFrozen` -> `codes.Unauthenticated`
- `ErrRefreshTokenInvalid` -> `codes.Unauthenticated`
- `ErrRefreshTokenRevoked` -> `codes.Unauthenticated`

不改 `pkg/grpcutil/errors.go`，继续复用 `grpcutil.ToGRPCError()`

### 7. 注册与装配

修改 `internal/domains/identity/grpc/register.go`：
- 签名改为 `RegisterServices(reg, coreDB, rdb, authCfg)`
- 构造 `UserRepo`
- 构造 `RefreshTokenRepo`
- 读取 PEM，构造 `RS256Signer`
- 构造 `AuthService`
- 注册 `identity.v1.AuthService`

修改 `cmd/monolith/main.go`：
- 把 `rdb` 和 `cfg.Auth` 传入 `identitygrpc.RegisterServices`
- 在 interceptor 链中插入 auth context interceptor

### 8. 后端身份注入 Interceptor

新增 `internal/middleware/auth_context.go`：
- `UnaryAuthContextInterceptor()`
- 从 gRPC metadata 读取 `x-user-id` / `x-tenant-id` / `x-user-role`
- 写入 context

新增 `pkg/identityctx/identityctx.go`：
- `UserID(ctx)`, `TenantID(ctx)`, `Role(ctx)`
- 也可补 `MustUserID(ctx)` 这类 helper，但先保持最小集

白名单至少包含：
- `"/parkhub.identity.v1.AuthService/Login"`
- `"/parkhub.identity.v1.AuthService/RefreshToken"`
- `"/parkhub.identity.v1.AuthService/Logout"`
- `"/parkhub.identity.v1.AuthService/GetJWKS"`
- `"/grpc.health.v1.Health/Check"`
- `"/grpc.health.v1.Health/Watch"`

安全前提要写进方案：
- 只有当 gRPC 端口不直接暴露给客户端时，才信任这些 header
- 如果未来需要允许外部直连 gRPC，就要改成后端自己验 access token，而不是信任 header

### 9. APISIX 配置

`configs/apisix/` 下新增或更新路由：
- 受保护路由启用 `jwt-auth`
- APISIX 使用静态 `public_key` 配置验签
- `proxy-rewrite` 将 claims 映射到 `X-User-Id` / `X-Tenant-Id` / `X-User-Role`
- `Login` / `RefreshToken` / `Logout` / `GetJWKS` 路由跳过 `jwt-auth`

说明：
- `GetJWKS` 只作为公钥导出与调试接口，不依赖 APISIX 动态拉取
- 如果后续确认 APISIX 版本和插件支持远端 JWKS，再单独演进

### 10. 验证

```bash
make proto-lint && make proto-gen
make test
make build-monolith
make docker-up
./bin/parkhub &

# 1. 直连 gRPC 测 Login
grpcurl -plaintext -d '{"username":"admin","password":"xxx"}' \
  localhost:50051 parkhub.identity.v1.AuthService/Login

# 2. 测 Refresh
#    第一次 Refresh 成功，第二次用同一个 refresh token 必须失败

# 3. 测 Logout
#    Logout 后再 Refresh 必须失败；重复 Logout 应幂等成功

# 4. 经 APISIX 访问受保护接口
curl localhost:9080/identity/.../GetUser -H "Authorization: Bearer <access>"

# 5. 验证后端 interceptor
#    业务方法里能从 ctx 读到 userID / tenantID / role

# 6. 验证安全边界
#    非网关来源若可直连 gRPC，不应被允许伪造身份 header
```

---

## 关键文件清单

**新增**:
- `api/proto/identity/v1/auth.proto`
- `internal/domains/identity/domain/{token.go,signer.go,rs256_signer.go}` + 测试
- `internal/domains/identity/repository/auth_interface.go`
- `internal/domains/identity/repository/cache/refresh_token_repo.go` + 测试
- `internal/domains/identity/service/{auth_interface.go,auth_service.go}` + 测试
- `internal/domains/identity/grpc/auth_server.go` + 测试
- `internal/middleware/auth_context.go` + 测试
- `pkg/identityctx/identityctx.go`
- `configs/keys/.gitignore`
- `configs/keys/README.md`
- APISIX 路由配置文件

**修改**:
- `go.mod` / `go.sum`
- `configs/config.yaml`
- `internal/config/config.go`
- `internal/domains/identity/errs/errors.go`
- `internal/domains/identity/grpc/register.go`
- `cmd/monolith/main.go`

**复用**:
- `bcrypt.CompareHashAndPassword`
- `repository.UserRepo.GetByUsername`
- `grpcutil.ToGRPCError`
- sms 域的 Redis cache 实现方式
- identity/sms 现有 mocks 和 grpc server 测试写法

## 非目标

这期不做：
- access token 黑名单
- 多设备会话管理
- APISIX 从后端动态拉 JWKS
- 外部客户端直连 gRPC 时的 access token 后端自验
