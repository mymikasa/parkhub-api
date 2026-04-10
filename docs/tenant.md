# Identity 域 - 租户服务实现计划

## Context

ParkHub SaaS 停车管理系统的第一个业务域实现。租户(Tenant)是整个系统的多租户隔离核心，所有其他域的数据都通过 `tenant_id` 关联。租户服务提供 CRUD 操作，由 PLATFORM_ADMIN 角色管理。此实现将建立所有后续域服务的代码模板。

## 开发模式

**TDD（测试驱动开发）**— 严格按 gox:grpc-service-dev 流程，每步先写测试再写实现，`go test` 通过后才进入下一步。

## 实现顺序（严格按序执行）

### Step 1 — Domain 实体定义

**新建** `internal/domains/identity/errs/errors.go`：
- `ErrTenantNotFound`, `ErrTenantAlreadyExists`, `ErrTenantInvalidStatus`

**新建** `internal/domains/identity/domain/tenant.go`：
- `TenantStatus` 枚举: `type TenantStatus string` + const (active/inactive/suspended)
- `PlanType` 枚举: `type PlanType string` + const (free/basic/pro/enterprise)
- `Tenant` struct: ID, Name, ContactName, ContactPhone, ContactEmail, Status, Address, PlanType, CreatedAt, UpdatedAt
- 工厂函数: `NewTenant(name, contactName, contactPhone, contactEmail, address string, planType PlanType) *Tenant`
- 业务方法: `IsActive()`, `Suspend()`, `Activate()`, `Deactivate()` — 含状态转换校验，返回 `errs.ErrTenantInvalidStatus`

**新建** `internal/domains/identity/domain/tenant_test.go`：
- 工厂函数正常/异常分支
- 状态转换方法（active→suspended ✓, inactive→suspended ✗）
- 边界值（空字符串等）

验证: `go test ./internal/domains/identity/domain/...`

### Step 2 — DAO 层

**新建** `internal/domains/identity/repository/dao/tenant.go`：
- GORM 模型 `Tenant` struct + gorm tags (varchar, primaryKey, uniqueIndex, autoCreateTime/autoUpdateTime)
- `TableName() = "tenants"`
- `TenantFilter` 结构体（Status, Keyword 等查询过滤）
- `TenantDAO` 接口: Insert, FindByID, FindByName, FindAll(filter, page, pageSize), Update, Delete
- `GORMTenantDAO` struct 持有 `*gorm.DB`，实现 `TenantDAO`
- 构造函数 `NewTenantDAO(db *gorm.DB) TenantDAO`
- 错误转换: `gorm.ErrRecordNotFound` → `errs.ErrTenantNotFound`, 唯一约束冲突 → `errs.ErrTenantAlreadyExists`
- `//go:generate mockgen -source=./tenant.go -package=daomocks -destination=./mocks/tenant.mock.go TenantDAO`

**新建** `internal/domains/identity/repository/dao/tenant_test.go`：
- SQLite 内存数据库集成测试
- `setupTestDB(t)` helper
- 覆盖: CRUD、NotFound 错误、唯一约束冲突、过滤条件、分页边界

验证: `go test ./internal/domains/identity/repository/dao/...`

生成 mock: `cd internal/domains/identity/repository/dao && go generate ./tenant.go`

### Step 3 — Repository 层

**新建** `internal/domains/identity/repository/interface.go`：
- `TenantFilter` 结构体（使用 domain 类型）
- `TenantRepo` 接口: Create, GetByID, GetByName, List(filter, page, pageSize), Update, Delete（参数/返回 `*domain.Tenant`）
- `//go:generate mockgen -source=./interface.go -package=repomocks -destination=./mocks/repo.mock.go TenantRepo`

**新建** `internal/domains/identity/repository/tenant_repo.go`：
- 私有 `tenantRepo` struct 持有 `dao.TenantDAO`
- `NewTenantRepo(d dao.TenantDAO) TenantRepo` 构造函数
- `toDomain(d *dao.Tenant) *domain.Tenant` — dao → domain 转换
- `toEntity(u *domain.Tenant) *dao.Tenant` — domain → dao 转换

**新建** `internal/domains/identity/repository/tenant_repo_test.go`：
- 使用 mock DAO 纯单元测试
- 覆盖: toDomain/toEntity 转换正确性、DAO 错误映射

验证: `go test ./internal/domains/identity/repository/...`

生成 mock: `cd internal/domains/identity/repository && go generate ./interface.go`

### Step 4 — Service 层

**新建** `internal/domains/identity/service/interface.go`：
- `CreateTenantRequest`, `UpdateTenantRequest`, `ListTenantsRequest` 请求结构体
- `TenantListResponse` 响应结构体（Tenants, Total, Page, PageSize, TotalPages）
- `TenantService` 接口: CreateTenant, GetTenant, ListTenants, UpdateTenant, DeleteTenant

**新建** `internal/domains/identity/service/tenant_service.go`：
- 私有 `tenantService` struct 持有 `repository.TenantRepo`
- `NewTenantService(repo repository.TenantRepo) TenantService` 构造函数
- CreateTenant: 生成 UUID、设默认 status=active/plan_type=free、调用 repo.Create
- UpdateTenant: 先 GetByID、应用变更、校验状态转换、调用 repo.Update
- ListTenants: 默认 page=1/pageSize=20、上限 100、计算 totalPages
- `//go:generate mockgen -source=./tenant_service.go -package=servicemocks -destination=./mocks/tenant_service.mock.go TenantService`

**新建** `internal/domains/identity/service/tenant_service_test.go`：
- 使用 mock Repo 纯单元测试
- 覆盖: 正常 CRUD、唯一性校验、业务规则、边界条件

验证: `go test ./internal/domains/identity/service/...`

**依赖**: `github.com/google/uuid`

### Step 5 — Proto 定义

**新建** `api/proto/identity/v1/tenant.proto`：
- `package parkhub.identity.v1`
- `option go_package = "github.com/parkhub/api/internal/gen/api/proto/identity/v1;identityv1"`
- import: `common/v1/pagination.proto`, `google/protobuf/timestamp.proto`
- 枚举: `TenantStatus`, `PlanType`（UNSPECIFIED=0 开头）
- 消息: `Tenant` (tenant_id, name, contact_name, contact_phone, contact_email, status, address, plan_type, created_at, updated_at)
- 请求/响应: CreateTenant, GetTenant, ListTenants (用 common.v1.Pagination), UpdateTenant, DeleteTenant
- 服务: `TenantService` 定义 5 个 RPC
- ID 字段命名: `tenant_id`（遵循 gox 约定）

生成: `buf generate`

### Step 6 — gRPC Server

**新建** `internal/domains/identity/grpc/tenant_server.go`：
- `TenantGRPCServer` struct: 嵌入 `identityv1.UnimplementedTenantServiceServer`，持有 `service.TenantService`
- `NewTenantGRPCServer(svc service.TenantService) *TenantGRPCServer` 构造函数
- 实现 5 个 RPC 方法：提取 proto request → 调用 service → domain→proto 转换 → 返回
- `toGRPCError(err error) error` — 使用 `grpcutil.ToGRPCError()` + 错误映射表

**新建** `internal/domains/identity/grpc/helpers.go`：
- `toProtoTenant(t *domain.Tenant) *identityv1.Tenant`
- `domainStatusToProto` / `domainStatusFromProto`
- `domainPlanToProto` / `domainPlanFromProto`

**新建** `internal/domains/identity/grpc/tenant_server_test.go`：
- bufconn in-process gRPC 端到端测试
- `setupTestServer(t)` 返回 `identityv1.TenantServiceClient`
- 覆盖: 每个 RPC 正常路径、错误映射、分页默认值、转换函数

验证: `go test ./internal/domains/identity/grpc/...`

### Step 7 — 注册到 Registry

**新建** `internal/domains/identity/grpc/register.go`：
```go
func RegisterServices(reg *registry.Registry, coreDB *gorm.DB) {
    // DAO
    tenantDAO := dao.NewTenantDAO(coreDB)
    // Repository
    tenantRepo := repository.NewTenantRepo(tenantDAO)
    // Service
    tenantSvc := service.NewTenantService(tenantRepo)
    // gRPC Server
    reg.MustRegister("identity.v1.TenantService", func(s *grpc.Server) {
        identityv1.RegisterTenantServiceServer(s, NewTenantGRPCServer(tenantSvc))
    })
}
```

**修改** `cmd/monolith/main.go`：
- 创建 `registry.New()` 实例
- AutoMigrate 添加 `&dao.Tenant{}`
- 调用 `identitygrpc.RegisterServices(reg, db)`
- 调用 `reg.RegisterAll(s)`

### 依赖管理

```bash
go get github.com/google/uuid
go get go.uber.org/mock/mockgen@latest
go get gorm.io/driver/sqlite  # 仅测试用
```

## 文件清单

| 操作 | 文件路径 |
|------|---------|
| 新建 | `internal/domains/identity/errs/errors.go` |
| 新建 | `internal/domains/identity/domain/tenant.go` |
| 新建 | `internal/domains/identity/domain/tenant_test.go` |
| 新建 | `internal/domains/identity/repository/dao/tenant.go` |
| 新建 | `internal/domains/identity/repository/dao/tenant_test.go` |
| 新建 | `internal/domains/identity/repository/dao/mocks/tenant.mock.go` (生成) |
| 新建 | `internal/domains/identity/repository/interface.go` |
| 新建 | `internal/domains/identity/repository/tenant_repo.go` |
| 新建 | `internal/domains/identity/repository/tenant_repo_test.go` |
| 新建 | `internal/domains/identity/repository/mocks/repo.mock.go` (生成) |
| 新建 | `internal/domains/identity/service/interface.go` |
| 新建 | `internal/domains/identity/service/tenant_service.go` |
| 新建 | `internal/domains/identity/service/tenant_service_test.go` |
| 新建 | `internal/domains/identity/service/mocks/tenant_service.mock.go` (生成) |
| 新建 | `api/proto/identity/v1/tenant.proto` |
| 新建 | `internal/domains/identity/grpc/tenant_server.go` |
| 新建 | `internal/domains/identity/grpc/helpers.go` |
| 新建 | `internal/domains/identity/grpc/tenant_server_test.go` |
| 新建 | `internal/domains/identity/grpc/register.go` |
| 修改 | `cmd/monolith/main.go` — Registry + AutoMigrate + 注册服务 |
| 生成 | `internal/gen/api/proto/identity/v1/tenant.pb.go` (buf generate) |
| 生成 | `internal/gen/api/proto/identity/v1/tenant_grpc.pb.go` (buf generate) |

## 关键设计决策

- **Tenant 本身无 tenant_id** — 它就是租户实体，其他域通过 tenant_id 引用它
- **硬删除** — 当前阶段直接删除，后续可改为软删除
- **不使用 Wire** — 首个服务手动组装，后续引入
- **Registry 模式** — `RegisterServices(reg, coreDB)` 通过 Registry 注册，与 gox 约定一致

## 验证清单

每步完成后执行：
- [ ] `go vet ./internal/domains/identity/...` 无错误
- [ ] `go build ./internal/domains/identity/...` 编译通过
- [ ] `go test ./internal/domains/identity/...` 全部通过

最终集成验证：
```bash
buf lint && buf generate
go build ./cmd/monolith
go vet ./...
go test ./...
```
