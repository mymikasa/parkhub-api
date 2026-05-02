# SMS 服务 TDD 开发计划

## 整体架构

```
internal/domains/sms/
  domain/           # 纯领域模型 + 领域错误
  errs/             # sentinel errors
  gateway/          # SmsGateway 接口 + mock 实现
    mocks/
  repository/       # repository 接口 + 聚合 cache/dao
    mocks/
    dao/            # GORM 实体
      mocks/
    cache/          # Redis 操作抽象
      mocks/
  service/          # 业务逻辑
    mocks/
  grpc/             # gRPC delivery

api/proto/sms/v1/   # Protobuf 定义
```

## 关键约束

- SMS 数据先落入 **当前 monolith 指向的单库**，不新增独立 `parkhub_sms` 数据库
- 保持现有分层：`service -> repository -> dao/cache/gateway`
- 复用并迁移现有 `internal/domains/identity/domain/sms.go`，最终收敛到 `internal/domains/sms/domain/sms_code.go`
- 验证码采用传统语义：**发送成功后才成为可验证验证码**；网关失败时只记录失败审计，不保留可用于验证的 Redis 验证码

## TDD 流程：6 个迭代，每轮 Red -> Green -> Refactor

---

### Phase 1: Domain Layer（纯单元测试，无依赖）

**先写测试 -> 再写实现**

| 测试文件 | 测试内容 |
|----------|---------|
| `domain/sms_code_test.go` | `NewSmsCode()` 工厂函数：验证 Code 长度 6 位、过期时间正确设置、Purpose 合法值 |
| `domain/sms_code_test.go` | `Verify()` 方法：正确的 code 返回 nil / 错误 code 返回 ErrCodeMismatch / 已使用返回 ErrCodeUsed / 已过期返回 ErrCodeExpired |
| `domain/sms_code_test.go` | `IsExpired()` 方法：过期/未过期判断正确 |
| `errs/errors_test.go` | 验证所有 sentinel error 变量已定义（编译期保障） |

**领域模型设计**：

```go
// domain/sms_code.go
type SmsCode struct {
    ID, Phone, Code, Purpose string
    ExpiresAt time.Time
    Used      bool
    CreatedAt time.Time
}

type SmsPurpose string // "login", "register", "reset_password"

func NewSmsCode(phone string, purpose SmsPurpose, ttl time.Duration) *SmsCode
func (s *SmsCode) Verify(input string) error
func (s *SmsCode) IsExpired() bool
```

**领域错误**：

```go
// errs/errors.go
var (
    ErrCodeNotFound   = errors.New("sms code not found")
    ErrCodeExpired    = errors.New("sms code expired")
    ErrCodeUsed       = errors.New("sms code already used")
    ErrCodeMismatch   = errors.New("sms code mismatch")
    ErrPhoneRateLimit = errors.New("phone rate limit exceeded")
    ErrInvalidPhone   = errors.New("invalid phone number")
    ErrInvalidPurpose = errors.New("invalid sms purpose")
)
```

---

### Phase 2: Gateway Layer（SmsGateway 接口 + Mock 实现）

**先写测试 -> 再写实现**

| 测试文件 | 测试内容 |
|----------|---------|
| `gateway/mock_gateway_test.go` | MockSmsGateway 返回 nil 表示发送成功 / 返回 error 表示发送失败 |

**接口定义**：

```go
// gateway/interface.go
type SmsGateway interface {
    Send(ctx context.Context, phone, code, purpose string) error
}

// gateway/mock_gateway.go — 记录调用次数和参数，用于测试验证
type MockSmsGateway struct { calls []CallRecord }
```

---

### Phase 3: Repository Layer — Cache (Redis) + DAO (MySQL)

#### 3A. Redis Cache — 先写测试

| 测试文件 | 测试内容 |
|----------|---------|
| `cache/sms_cache_test.go` | `Store()` 存储验证码 + `Retrieve()` 取出并反序列化正确 |
| `cache/sms_cache_test.go` | `Retrieve()` 对不存在的 key 返回 `ErrCodeNotFound` |
| `cache/sms_cache_test.go` | `SetRateLimit()` / `CheckRateLimit()` 频率限制生效（同一手机号 60s 内限制） |
| `cache/sms_cache_test.go` | `MarkUsed()` 标记已使用后再次 Verify 失败 |

> Redis 测试策略：使用 `miniredis` 库（内存 Redis mock），零外部依赖

**接口定义**：

```go
// cache/interface.go
type SmsCache interface {
    Store(ctx context.Context, code *domain.SmsCode) error
    Retrieve(ctx context.Context, phone, purpose string) (*domain.SmsCode, error)
    MarkUsed(ctx context.Context, phone, purpose string) error
    SetRateLimit(ctx context.Context, phone string, ttl time.Duration) error
    CheckRateLimit(ctx context.Context, phone string) (bool, error)
}
```

#### 3B. MySQL DAO — 先写测试

| 测试文件 | 测试内容 |
|----------|---------|
| `dao/sms_record_test.go` | `Insert()` 正常插入发送记录 |
| `dao/sms_record_test.go` | `FindByID()` / `FindByPhone()` 查询正确 |
| `dao/sms_record_test.go` | `List()` 分页 + 按手机号/时间段过滤 |

> 使用与 identity 域相同的 SQLite in-memory 测试策略

#### 3C. Repository 聚合层 — 先写测试

| 测试文件 | 测试内容 |
|----------|---------|
| `repository/sms_repo_test.go` | `SaveCode()` 正常写入 Redis 验证码 + MySQL 审计记录 |
| `repository/sms_repo_test.go` | `SaveSendFailure()` 仅写入 MySQL 失败审计，不写可验证验证码 |
| `repository/sms_repo_test.go` | `GetCode()` / `MarkCodeUsed()` 正确委托 cache |
| `repository/sms_repo_test.go` | `CheckRateLimit()` / `SetRateLimit()` 正确委托 cache |

**接口定义**：

```go
// repository/interface.go
type SmsRepository interface {
    SaveCode(ctx context.Context, code *domain.SmsCode) error
    SaveSendFailure(ctx context.Context, phone string, purpose domain.SmsPurpose, providerErr string) error
    GetCode(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error)
    MarkCodeUsed(ctx context.Context, phone string, purpose domain.SmsPurpose) error
    SetRateLimit(ctx context.Context, phone string, ttl time.Duration) error
    CheckRateLimit(ctx context.Context, phone string) (bool, error)
}
```

---

### Phase 4: Service Layer（核心业务逻辑）

**先写测试 -> 再写实现**

| 测试文件 | 测试内容 |
|----------|---------|
| `service/sms_service_test.go` | `SendCode()` 正常流程：校验手机号/用途 -> 限流检查 -> 生成 code -> 调网关发送 -> repository 保存验证码 + 审计记录 + 设置限流 |
| `service/sms_service_test.go` | `SendCode()` 频率限制：60s 内重复发送返回 `ErrPhoneRateLimit` |
| `service/sms_service_test.go` | `SendCode()` 手机号校验：非法号码返回 `ErrInvalidPhone` |
| `service/sms_service_test.go` | `SendCode()` 网关失败：SmsGateway 返回 error 时，仅记录失败审计，不缓存可验证验证码，也不设置限流 |
| `service/sms_service_test.go` | `VerifyCode()` 正确 code 验证通过 + 标记已使用 |
| `service/sms_service_test.go` | `VerifyCode()` 错误 code 返回 `ErrCodeMismatch` |
| `service/sms_service_test.go` | `VerifyCode()` 过期 code 返回 `ErrCodeExpired` |
| `service/sms_service_test.go` | `VerifyCode()` 已使用 code 返回 `ErrCodeUsed` |

**mock 依赖**：`SmsRepository` mock + `SmsGateway` mock

**接口定义**：

```go
// service/interface.go
type SmsService interface {
    SendCode(ctx context.Context, phone, purpose string) error
    VerifyCode(ctx context.Context, phone, code, purpose string) error
}
```

---

### Phase 5: gRPC Layer + Proto 定义

#### 5A. 先写 Proto -> `make proto-gen`

```protobuf
// api/proto/sms/v1/sms.proto
package parkhub.sms.v1;

service SmsService {
  rpc SendCode(SendCodeRequest) returns (SendCodeResponse);
  rpc VerifyCode(VerifyCodeRequest) returns (VerifyCodeResponse);
}
```

#### 5B. 先写测试 -> 再写实现

| 测试文件 | 测试内容 |
|----------|---------|
| `grpc/sms_server_test.go` | `SendCode` RPC 正常返回 OK |
| `grpc/sms_server_test.go` | `SendCode` RPC 频率限制映射到 `ResourceExhausted` |
| `grpc/sms_server_test.go` | `SendCode` RPC 无效手机号映射到 `InvalidArgument` |
| `grpc/sms_server_test.go` | `VerifyCode` RPC 正常返回 OK |
| `grpc/sms_server_test.go` | `VerifyCode` RPC 错误 code 映射到 `Unauthenticated` |
| `grpc/sms_server_test.go` | `VerifyCode` RPC 过期映射到 `DeadlineExceeded` |

> 使用 `bufconn` in-process gRPC 测试，与 identity 域一致

**错误映射**：

```go
var smsErrorMappings = []grpcutil.ErrorMapping{
    {Err: errs.ErrCodeNotFound, Code: codes.NotFound},
    {Err: errs.ErrCodeExpired, Code: codes.DeadlineExceeded},
    {Err: errs.ErrCodeUsed, Code: codes.FailedPrecondition},
    {Err: errs.ErrCodeMismatch, Code: codes.Unauthenticated},
    {Err: errs.ErrPhoneRateLimit, Code: codes.ResourceExhausted},
    {Err: errs.ErrInvalidPhone, Code: codes.InvalidArgument},
    {Err: errs.ErrInvalidPurpose, Code: codes.InvalidArgument},
}
```

---

### Phase 6: 集成注册 + 基础设施

| 任务 | 说明 |
|------|------|
| `grpc/register.go` | 仿照 identity 的 `RegisterServices(reg, db, rdb)` 组装依赖链，service 依赖 repository，repository 聚合 cache + dao |
| `internal/config/config.go` | 增加 Redis 配置结构与环境变量读取 |
| `cmd/monolith/main.go` | 添加 Redis 连接 + `smsgrpc.RegisterServices(reg, db, rdb)` |
| `scripts/init-databases.sql` | **不新增** `parkhub_sms`，继续使用 monolith 当前指向数据库 |
| `docker-compose.yml` | 确认 Redis 服务可用 |
| 运行 `make test` | 全量测试通过 |

---

## TDD 循环纪律

每个 Phase 严格遵循：

```
1. RED     -> 写测试，确认编译失败（接口/类型不存在）
2. GREEN   -> 写最小实现，让测试通过
3. REFACTOR -> 重构，确保测试仍通过
4. make lint && make test  -> 全量检查
```

## 每层 mock 生成命令

```bash
# Phase 3A - Cache mock
mockgen -source=internal/domains/sms/repository/cache/interface.go \
  -package=cachemocks -destination=internal/domains/sms/repository/cache/mocks/sms_cache.mock.go SmsCache

# Phase 3B - DAO mock
mockgen -source=internal/domains/sms/repository/dao/sms_record.go \
  -package=daomocks -destination=internal/domains/sms/repository/dao/mocks/sms_record.mock.go SmsRecordDAO

# Phase 3C - Repository mock
mockgen -source=internal/domains/sms/repository/interface.go \
  -package=repomocks -destination=internal/domains/sms/repository/mocks/repo.mock.go SmsRepository

# Phase 4 - Service mock
mockgen -source=internal/domains/sms/service/interface.go \
  -package=servicemocks -destination=internal/domains/sms/service/mocks/sms_service.mock.go SmsService
```

## 需要新增的依赖

| 依赖 | 用途 |
|------|------|
| `github.com/alicebob/miniredis/v2` | Redis in-memory mock（替代真实 Redis 做单元测试）|
| `github.com/redis/go-redis/v9` | Redis 客户端（生产环境连接） |

## 可直接开工的文件清单

### 1. 新建领域层

- `internal/domains/sms/errs/errors.go`
- `internal/domains/sms/errs/errors_test.go`
- `internal/domains/sms/domain/sms_code.go`
- `internal/domains/sms/domain/sms_code_test.go`

### 2. 新建 gateway 层

- `internal/domains/sms/gateway/interface.go`
- `internal/domains/sms/gateway/mock_gateway.go`
- `internal/domains/sms/gateway/mock_gateway_test.go`

### 3. 新建 repository 子层

- `internal/domains/sms/repository/cache/interface.go`
- `internal/domains/sms/repository/cache/redis_cache.go`
- `internal/domains/sms/repository/cache/sms_cache_test.go`
- `internal/domains/sms/repository/cache/mocks/sms_cache.mock.go`
- `internal/domains/sms/repository/dao/sms_record.go`
- `internal/domains/sms/repository/dao/sms_record_test.go`
- `internal/domains/sms/repository/dao/mocks/sms_record.mock.go`
- `internal/domains/sms/repository/interface.go`
- `internal/domains/sms/repository/sms_repo.go`
- `internal/domains/sms/repository/sms_repo_test.go`
- `internal/domains/sms/repository/mocks/repo.mock.go`

### 4. 新建 service 层

- `internal/domains/sms/service/interface.go`
- `internal/domains/sms/service/sms_service.go`
- `internal/domains/sms/service/sms_service_test.go`
- `internal/domains/sms/service/mocks/sms_service.mock.go`

### 5. 新建 gRPC 层

- `api/proto/sms/v1/sms.proto`
- `internal/domains/sms/grpc/helpers.go`
- `internal/domains/sms/grpc/sms_server.go`
- `internal/domains/sms/grpc/sms_server_test.go`
- `internal/domains/sms/grpc/register.go`

### 6. 修改现有基础设施

- `internal/config/config.go`
- `cmd/monolith/main.go`
- `scripts/init-databases.sql`

### 7. 删除或迁移旧模型

- `internal/domains/identity/domain/sms.go`
  迁移完成后删除，引用方统一切到 `internal/domains/sms/domain`

## 最小接口草图

以下草图的目标不是一次性定死实现，而是让每层职责在开工前不再含糊。

### Domain

```go
package domain

import "time"

type SmsPurpose string

const (
    SmsPurposeLogin         SmsPurpose = "login"
    SmsPurposeRegister      SmsPurpose = "register"
    SmsPurposeResetPassword SmsPurpose = "reset_password"
)

type SmsCode struct {
    ID        string
    Phone     string
    Code      string
    Purpose   SmsPurpose
    ExpiresAt time.Time
    Used      bool
    CreatedAt time.Time
}

func NewSmsCode(phone string, purpose SmsPurpose, ttl time.Duration) (*SmsCode, error)
func (s *SmsCode) Verify(input string) error
func (s *SmsCode) MarkUsed()
func (s *SmsCode) IsExpired(now time.Time) bool
```

### Gateway

```go
package gateway

import "context"

type SmsGateway interface {
    Send(ctx context.Context, phone, code string, purpose string) error
}
```

### Cache

```go
package cache

import (
    "context"
    "time"

    "github.com/parkhub/api/internal/domains/sms/domain"
)

type SmsCache interface {
    Store(ctx context.Context, code *domain.SmsCode) error
    Retrieve(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error)
    MarkUsed(ctx context.Context, phone string, purpose domain.SmsPurpose) error
    SetRateLimit(ctx context.Context, phone string, ttl time.Duration) error
    CheckRateLimit(ctx context.Context, phone string) (bool, error)
}
```

### DAO

```go
package dao

import "context"

type SmsSendStatus string

const (
    SmsSendStatusSuccess SmsSendStatus = "success"
    SmsSendStatusFailed  SmsSendStatus = "failed"
)

type SmsRecord struct {
    ID          string `gorm:"primaryKey;type:varchar(36)"`
    Phone       string `gorm:"type:varchar(20);index"`
    Purpose     string `gorm:"type:varchar(32);index"`
    Code        string `gorm:"type:varchar(10)"`
    Status      string `gorm:"type:varchar(20);index"`
    ProviderErr *string `gorm:"type:text"`
    CreatedAt   int64  `gorm:"autoCreateTime:milli;index"`
}

func (SmsRecord) TableName() string

type SmsRecordFilter struct {
    Phone     string
    Purpose   string
    Status    string
    StartTime *int64
    EndTime   *int64
}

type SmsRecordDAO interface {
    Insert(ctx context.Context, record *SmsRecord) error
    FindByID(ctx context.Context, id string) (*SmsRecord, error)
    FindByPhone(ctx context.Context, phone string, page, pageSize int) ([]*SmsRecord, int64, error)
    List(ctx context.Context, filter SmsRecordFilter, page, pageSize int) ([]*SmsRecord, int64, error)
}
```

### Repository

```go
package repository

import (
    "context"
    "time"

    "github.com/parkhub/api/internal/domains/sms/domain"
)

type SmsRepository interface {
    SaveCode(ctx context.Context, code *domain.SmsCode) error
    SaveSendFailure(ctx context.Context, phone string, purpose domain.SmsPurpose, providerErr string) error
    GetCode(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error)
    MarkCodeUsed(ctx context.Context, phone string, purpose domain.SmsPurpose) error
    SetRateLimit(ctx context.Context, phone string, ttl time.Duration) error
    CheckRateLimit(ctx context.Context, phone string) (bool, error)
}
```

### Service

```go
package service

import (
    "context"

    "github.com/parkhub/api/internal/domains/sms/domain"
)

type SendCodeRequest struct {
    Phone   string
    Purpose domain.SmsPurpose
}

type VerifyCodeRequest struct {
    Phone   string
    Code    string
    Purpose domain.SmsPurpose
}

//go:generate mockgen -source=./interface.go -package=servicemocks -destination=./mocks/sms_service.mock.go SmsService

type SmsService interface {
    SendCode(ctx context.Context, req *SendCodeRequest) error
    VerifyCode(ctx context.Context, req *VerifyCodeRequest) error
}
```

### gRPC Register

```go
func RegisterServices(reg *registry.Registry, coreDB *gorm.DB, rdb *redis.Client) {
    smsDAO := dao.NewSmsRecordDAO(coreDB)
    smsCache := cache.NewRedisSmsCache(rdb)
    smsRepo := repository.NewSmsRepository(smsDAO, smsCache)
    smsGateway := gateway.NewMockSmsGateway()
    smsSvc := service.NewSmsService(smsRepo, smsGateway)

    reg.MustRegister("sms.v1.SmsService", func(s *grpc.Server) {
        smsv1.RegisterSmsServiceServer(s, grpc.NewSMSGRPCServer(smsSvc))
    })
}
```

## 建议实施顺序

1. 先完成 `domain + errs`，顺手迁移 `internal/domains/identity/domain/sms.go` 的引用。
2. 再做 `cache` 和 `dao`，把最底层存储能力跑通。
3. 接着补 `repository`，把“成功保存验证码”和“失败只记审计”两个分支钉死。
4. 然后实现 `service`，这里集中处理手机号校验、用途校验、限流和发送流程。
5. 最后接 `proto + grpc + register + main/config`，把基础设施一次性串起来。

## 执行顺序总结

```
Phase 1: domain/      (纯单元测试，5 min)
Phase 2: gateway/     (接口 + mock，5 min)
Phase 3A: cache/      (Redis 层测试 + 实现，15 min)
Phase 3B: dao/        (MySQL DAO 测试 + 实现，10 min)
Phase 3C: repository/ (聚合层测试 + 实现，10 min)
Phase 4: service/     (业务逻辑测试 + 实现，20 min)  <- 核心
Phase 5: proto + grpc (Proto + gRPC 测试 + 实现，15 min)
Phase 6: 集成注册      (注册 + main.go + SQL，10 min)
```

每个 Phase 内部严格 **先写测试再写实现**，确保代码覆盖率从内层向外层递减（测试金字塔）。
