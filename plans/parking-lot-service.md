# Plan: parking 域 ParkingLotService

## Context

parking 域当前为空白——无 proto 定义、无代码目录、无数据库表。`parkhub_parking` 数据库已在 Docker 初始化脚本中创建。本计划参照 identity 域的分层模式（proto → domain → dao → repo → service → grpc → register），按层横向推进，完成 ParkingLot 的 CRUD + GetStats。

**实体定义**:

```go
type ParkingLot struct {
    ID              string
    TenantID        string
    Name            string
    Address         string
    TotalSpaces     int
    AvailableSpaces int
    LotType         LotType           // underground | ground | stereo
    Status          ParkingLotStatus  // active | inactive
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**用户决策**:
- AvailableSpaces 创建时 = TotalSpaces，后续由 IoT 域事件或手动更新
- tenantID 由 gRPC Handler 从 context 提取，显式传给 Service（便于单测）
- 迁移策略沿用 GORM AutoMigrate
- GetStats 返回租户维度的聚合统计

**GetStats 响应**:
```go
type ParkingLotStatsResponse struct {
    TotalSpaces      int64
    AvailableSpaces  int64
    OccupiedVehicles int64
    TotalGates       int64   // IoT 域集成前暂返回 0
}
```

---

## 实现步骤

### Phase 1: Proto 定义 + 代码生成

新建 `api/proto/parking/v1/parking_lot.proto`，包名 `parkhub.parking.v1`。

```proto
syntax = "proto3";

package parkhub.parking.v1;

option go_package = "github.com/parkhub/api/internal/gen/api/proto/parking/v1;parkingv1";

import "common/v1/pagination.proto";
import "google/protobuf/timestamp.proto";

enum LotType {
  LOT_TYPE_UNSPECIFIED = 0;
  LOT_TYPE_UNDERGROUND = 1;
  LOT_TYPE_GROUND = 2;
  LOT_TYPE_STEREO = 3;
}

enum ParkingLotStatus {
  PARKING_LOT_STATUS_UNSPECIFIED = 0;
  PARKING_LOT_STATUS_ACTIVE = 1;
  PARKING_LOT_STATUS_INACTIVE = 2;
}

message ParkingLot {
  string id = 1;
  string tenant_id = 2;
  string name = 3;
  string address = 4;
  int32 total_spaces = 5;
  int32 available_spaces = 6;
  LotType lot_type = 7;
  ParkingLotStatus status = 8;
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

message CreateParkingLotRequest {
  string name = 1;
  string address = 2;
  int32 total_spaces = 3;
  LotType lot_type = 4;
}

message CreateParkingLotResponse {
  ParkingLot parking_lot = 1;
}

message GetParkingLotRequest {
  string id = 1;
}

message GetParkingLotResponse {
  ParkingLot parking_lot = 1;
}

message ListParkingLotsRequest {
  ParkingLotStatus status = 1;
  LotType lot_type = 2;
  string keyword = 3;
  common.v1.PaginationRequest pagination = 4;
}

message ListParkingLotsResponse {
  repeated ParkingLot parking_lots = 1;
  common.v1.PaginationResponse pagination = 2;
}

message UpdateParkingLotRequest {
  string id = 1;
  optional string name = 2;
  optional string address = 3;
  optional int32 total_spaces = 4;
  optional LotType lot_type = 5;
  optional ParkingLotStatus status = 6;
}

message UpdateParkingLotResponse {
  ParkingLot parking_lot = 1;
}

message DeleteParkingLotRequest {
  string id = 1;
}

message DeleteParkingLotResponse {}

message GetParkingLotStatsRequest {}

message GetParkingLotStatsResponse {
  int64 total_spaces = 1;
  int64 available_spaces = 2;
  int64 occupied_vehicles = 3;
  int64 total_gates = 4;
}

service ParkingLotService {
  rpc CreateParkingLot(CreateParkingLotRequest) returns (CreateParkingLotResponse);
  rpc GetParkingLot(GetParkingLotRequest) returns (GetParkingLotResponse);
  rpc ListParkingLots(ListParkingLotsRequest) returns (ListParkingLotsResponse);
  rpc UpdateParkingLot(UpdateParkingLotRequest) returns (UpdateParkingLotResponse);
  rpc DeleteParkingLot(DeleteParkingLotRequest) returns (DeleteParkingLotResponse);
  rpc GetParkingLotStats(GetParkingLotStatsRequest) returns (GetParkingLotStatsResponse);
}
```

运行 `make proto-lint && make proto-gen` 生成代码到 `internal/gen/api/proto/parking/v1/`。

> 本期不引入 `buf.validate` / protovalidate。输入校验放在 gRPC handler/service 层完成，避免修改 `buf.yaml`、`buf.gen.yaml` 和运行时校验链路。后续如果项目统一接入 protovalidate，再批量迁移 proto 注解。

---

### Phase 2: Domain 层

**2.1 领域模型** — `internal/domains/parking/domain/parking_lot.go`

```go
package domain

import "github.com/parkhub/api/internal/domains/parking/errs"

type LotType string

const (
    LotTypeUnderground LotType = "underground"
    LotTypeGround      LotType = "ground"
    LotTypeStereo      LotType = "stereo"
)

type ParkingLotStatus string

const (
    ParkingLotStatusActive   ParkingLotStatus = "active"
    ParkingLotStatusInactive ParkingLotStatus = "inactive"
)

type ParkingLot struct {
    ID              string
    TenantID        string
    Name            string
    Address         string
    TotalSpaces     int
    AvailableSpaces int
    LotType         LotType
    Status          ParkingLotStatus
    CreatedAt       int64  // 毫秒时间戳，与 identity 域一致
    UpdatedAt       int64
}

func NewParkingLot(name, address string, totalSpaces int, lotType LotType) *ParkingLot {
    return &ParkingLot{
        Name:            name,
        Address:         address,
        TotalSpaces:     totalSpaces,
        AvailableSpaces: totalSpaces,
        LotType:         lotType,
        Status:          ParkingLotStatusActive,
    }
}

func (l *ParkingLot) IsActive() bool {
    return l.Status == ParkingLotStatusActive
}

func (l *ParkingLot) Deactivate() error {
    if l.Status != ParkingLotStatusActive {
        return errs.ErrParkingLotInvalidStatus
    }
    l.Status = ParkingLotStatusInactive
    return nil
}

func (l *ParkingLot) Activate() error {
    if l.Status != ParkingLotStatusInactive {
        return errs.ErrParkingLotInvalidStatus
    }
    l.Status = ParkingLotStatusActive
    return nil
}
```

**2.2 哨兵错误** — `internal/domains/parking/errs/errors.go`

```go
package errs

import "errors"

var (
    ErrParkingLotNotFound        = errors.New("parking lot not found")
    ErrParkingLotAlreadyExists   = errors.New("parking lot already exists")
    ErrParkingLotInvalidStatus   = errors.New("invalid parking lot status transition")
    ErrParkingLotInvalidCapacity = errors.New("invalid parking lot capacity")
    ErrParkingLotNameDuplicate   = errors.New("parking lot name already exists under this tenant")
)
```

---

### Phase 3: DAO 层

**3.1 GORM 模型** — `internal/domains/parking/repository/dao/parking_lot.go`

```go
type ParkingLot struct {
    ID              string `gorm:"primaryKey;type:varchar(36)"`
    TenantID        string `gorm:"type:varchar(36);uniqueIndex:idx_tenant_name;index"`
    Name            string `gorm:"type:varchar(100);uniqueIndex:idx_tenant_name"`
    Address         string `gorm:"type:varchar(255)"`
    TotalSpaces     int    `gorm:"type:int"`
    AvailableSpaces int    `gorm:"type:int"`
    LotType         string `gorm:"type:varchar(20)"`
    Status          string `gorm:"type:varchar(20)"`
    CreatedAt       int64  `gorm:"autoCreateTime:milli"`
    UpdatedAt       int64  `gorm:"autoUpdateTime:milli"`
}
```

关键点：
- `TenantID` + `Name` 联合唯一索引 `idx_tenant_name`（同租户下车场名不重复）
- `TenantID` 单独索引（租户隔离查询加速）

**3.2 DAO 接口 + 实现**

```go
type ParkingLotFilter struct {
    TenantID string  // 必填，租户隔离
    Status   string
    LotType  string
    Keyword  string
}

type ParkingLotDAO interface {
    Insert(ctx context.Context, lot *ParkingLot) error
    FindByID(ctx context.Context, tenantID, id string) (*ParkingLot, error)
    FindAll(ctx context.Context, filter ParkingLotFilter, page, pageSize int) ([]*ParkingLot, int64, error)
    Update(ctx context.Context, lot *ParkingLot) error
    Delete(ctx context.Context, tenantID, id string) error
    SumStats(ctx context.Context, tenantID string) (totalSpaces, availableSpaces int64, err error)
}
```

关键实现细节：
- 所有查询都带 `WHERE tenant_id = ?` 条件（租户隔离）
- `Insert` 捕获 duplicate entry 错误 → `ErrParkingLotNameDuplicate`
- `FindByID` / `Delete` 双条件 `WHERE tenant_id = ? AND id = ?`
- `FindAll` 支持 status/lot_type/keyword 过滤，keyword 匹配 name 和 address
- `SumStats` 做 `SELECT SUM(total_spaces), SUM(available_spaces) WHERE tenant_id = ?`
- mockgen 指令：`//go:generate mockgen -source=./parking_lot.go -package=daomocks -destination=./mocks/parking_lot.mock.go ParkingLotDAO`

---

### Phase 4: Repository 层

**4.1 接口** — `internal/domains/parking/repository/interface.go`

```go
type ParkingLotFilter struct {
    TenantID string
    Status   domain.ParkingLotStatus
    LotType  domain.LotType
    Keyword  string
}

type ParkingLotRepo interface {
    Create(ctx context.Context, lot *domain.ParkingLot) error
    GetByID(ctx context.Context, tenantID, id string) (*domain.ParkingLot, error)
    List(ctx context.Context, filter ParkingLotFilter, page, pageSize int) ([]*domain.ParkingLot, int64, error)
    Update(ctx context.Context, lot *domain.ParkingLot) error
    Delete(ctx context.Context, tenantID, id string) error
    SumStats(ctx context.Context, tenantID string) (totalSpaces, availableSpaces int64, err error)
}
```

mockgen 指令：`//go:generate mockgen -source=./interface.go -package=repomocks -destination=./mocks/repo.mock.go ParkingLotRepo`

**4.2 实现** — `internal/domains/parking/repository/parking_lot_repo.go`

模式与 identity 域完全一致：
- `toDomain(*dao.ParkingLot) *domain.ParkingLot` — 做 string→枚举类型转换
- `toEntity(*domain.ParkingLot) *dao.ParkingLot` — 做枚举→string 转换
- 方法内做 DAO filter 与 domain filter 的转换

---

### Phase 5: Service 层

**5.1 接口 + DTO** — `internal/domains/parking/service/interface.go`

```go
type CreateParkingLotRequest struct {
    TenantID    string
    Name        string
    Address     string
    TotalSpaces int
    LotType     domain.LotType
}

type UpdateParkingLotRequest struct {
    ID              string
    TenantID        string
    Name            *string
    Address         *string
    TotalSpaces     *int
    LotType         *domain.LotType
    Status          *domain.ParkingLotStatus
}

type ListParkingLotsRequest struct {
    TenantID string
    Status   domain.ParkingLotStatus
    LotType  domain.LotType
    Keyword  string
    Page     int
    PageSize int
}

type ParkingLotListResponse struct {
    ParkingLots []*domain.ParkingLot
    Total       int64
    Page        int
    PageSize    int
    TotalPages  int
}

type ParkingLotStatsResponse struct {
    TotalSpaces      int64
    AvailableSpaces  int64
    OccupiedVehicles int64
    TotalGates       int64
}

type ParkingLotService interface {
    Create(ctx context.Context, req *CreateParkingLotRequest) (*domain.ParkingLot, error)
    GetByID(ctx context.Context, tenantID, id string) (*domain.ParkingLot, error)
    List(ctx context.Context, req *ListParkingLotsRequest) (*ParkingLotListResponse, error)
    Update(ctx context.Context, req *UpdateParkingLotRequest) (*domain.ParkingLot, error)
    Delete(ctx context.Context, tenantID, id string) error
    GetStats(ctx context.Context, tenantID string) (*ParkingLotStatsResponse, error)
}
```

mockgen 指令：`//go:generate mockgen -source=./interface.go -package=servicemocks -destination=./mocks/parking_lot_service.mock.go ParkingLotService`

**5.2 实现** — `internal/domains/parking/service/parking_lot_service.go`

关键业务逻辑：

**Create**:
1. `domain.NewParkingLot(name, address, totalSpaces, lotType)` 构造领域对象
2. `uuid.New().String()` 生成 ID
3. 设置 `TenantID`
4. `repo.Create(ctx, lot)` 持久化
5. 重复名称由 DAO 层联合唯一索引兜底 → `ErrParkingLotNameDuplicate`

**GetByID**:
1. `repo.GetByID(ctx, tenantID, id)`，租户隔离由 DAO 的双条件保证

**List**:
1. page/pageSize 规范化（默认 1/20，上限 100）
2. 构造 `repository.ParkingLotFilter`
3. `repo.List(ctx, filter, page, pageSize)`
4. 计算 `totalPages = ceil(total / pageSize)`

**Update**:
1. `repo.GetByID(ctx, tenantID, id)` 获取当前实体
2. 逐字段检查指针参数，非 nil 则覆盖
3. Status 变更走领域方法 `lot.Activate()` / `lot.Deactivate()`，保证状态转换合法性
4. `total_spaces` 变更必须维护容量不变量：
   - `occupied = old.TotalSpaces - old.AvailableSpaces`
   - `newTotalSpaces >= occupied`，否则返回 `ErrParkingLotInvalidCapacity`
   - `AvailableSpaces = newTotalSpaces - occupied`
   - 这样可保证 `0 <= AvailableSpaces <= TotalSpaces`，`GetStats` 不会出现负占用
5. `repo.Update(ctx, lot)`

**Delete**:
1. 本期按物理删除实现：`repo.Delete(ctx, tenantID, id)`
2. 删除仍必须带 `tenantID` 条件，未命中返回 `ErrParkingLotNotFound`
3. 未来引入 Zone / Space / Event 后，如果存在历史关联，再新增归档/停用语义，不在本期混入

**GetStats**:
1. `repo.SumStats(ctx, tenantID)` 获取 totalSpaces / availableSpaces
2. `OccupiedVehicles = TotalSpaces - AvailableSpaces`
3. `TotalGates` 暂返回 0（IoT 域集成后对接）

---

### Phase 6: gRPC Handler + 注册

**6.1 Server** — `internal/domains/parking/grpc/parking_lot_server.go`

结构体嵌入 `parkingv1.UnimplementedParkingLotServiceServer`，持有 `service.ParkingLotService`。

每个 RPC 方法：
1. 从 `ctx` 提取 `tenantID`：`identityctx.TenantID(ctx)`，如果为空直接返回 `codes.Unauthenticated`
2. 转换 proto request → service DTO
3. 调用 service 方法
4. 转换 service response → proto response

错误映射：
```go
var errorMappings = []grpcutil.ErrorMapping{
    {Target: errs.ErrParkingLotNotFound,      Code: codes.NotFound},
    {Target: errs.ErrParkingLotNameDuplicate, Code: codes.AlreadyExists},
    {Target: errs.ErrParkingLotInvalidStatus, Code: codes.InvalidArgument},
    {Target: errs.ErrParkingLotInvalidCapacity, Code: codes.InvalidArgument},
}
```

输入校验：
- gRPC handler 校验必填字段：`tenantID`、`id`、`name`、`address`、`total_spaces > 0`、枚举不能为 `UNSPECIFIED`
- service 层重复兜底校验，防止绕过 gRPC handler 的单元测试或内部调用传入非法数据
- 缺少 `x-user-id` 仍由 `UnaryAuthContextInterceptor` 返回 `Unauthenticated`
- 缺少 `x-tenant-id` 由 parking handler 返回 `Unauthenticated`

**6.2 Helpers** — `internal/domains/parking/grpc/helpers.go`

- `toProtoParkingLot(*domain.ParkingLot) *parkingv1.ParkingLot`
- `domainLotTypeToProto(domain.LotType) parkingv1.LotType`
- `domainLotTypeFromProto(parkingv1.LotType) domain.LotType`
- `domainStatusToProto(domain.ParkingLotStatus) parkingv1.ParkingLotStatus`
- `domainStatusFromProto(parkingv1.ParkingLotStatus) domain.ParkingLotStatus`
- `toTimestamp(millis int64) *timestamppb.Timestamp`（复用 identity 域风格）

**6.3 注册** — `internal/domains/parking/grpc/register.go`

```go
func RegisterServices(reg *registry.Registry, db *gorm.DB) {
    lotDAO := dao.NewParkingLotDAO(db)
    lotRepo := repository.NewParkingLotRepo(lotDAO)
    lotSvc := service.NewParkingLotService(lotRepo)

    reg.MustRegister("parking.v1.ParkingLotService", func(s *grpc.Server) {
        parkingv1.RegisterParkingLotServiceServer(s, NewParkingLotGRPCServer(lotSvc))
    })
}
```

---

### Phase 7: 单体集成

**7.1 数据库连接**

当前 identity/sms 共用一个 `*gorm.DB` 连接 `parkhub_identity` 库。parking 域需要连 `parkhub_parking`。

方案：新增顶层 `parking_database` 配置，main.go 创建独立的 `*gorm.DB` 实例。这样不改动当前 `database` 字段语义，identity/sms 继续使用 `parkhub_identity`，parking 使用 `parkhub_parking`。

`configs/config.yaml` 新增：
```yaml
parking_database:
  host: localhost
  port: 3306
  user: parkhub
  password: parkhub
  dbname: parkhub_parking
```

`internal/config/config.go`：
- `Config` 顶层新增 ``ParkingDatabase DatabaseConfig `yaml:"parking_database"` ``
- `Default()` 中设置 `ParkingDatabase.DBName = "parkhub_parking"`
- `applyEnvOverrides()` 增加独立环境变量：`PARKING_DATABASE_URL`、`PARKING_DB_HOST`、`PARKING_DB_PORT`、`PARKING_DB_USER`、`PARKING_DB_PASSWORD`、`PARKING_DB_NAME`

**7.2 main.go 更新**

```go
// import
parkinggrpc "github.com/parkhub/api/internal/domains/parking/grpc"
parkingdao "github.com/parkhub/api/internal/domains/parking/repository/dao"

// 创建 parking DB
parkingDB, err := gorm.Open(mysql.Open(cfg.ParkingDatabase.DSN()), &gorm.Config{})

// AutoMigrate
parkingDB.AutoMigrate(&parkingdao.ParkingLot{})

// 注册
parkinggrpc.RegisterServices(reg, parkingDB)
```

**7.3 中间件**

`internal/middleware/auth_context.go` 无需修改——所有 parking RPC 都需要身份认证，无白名单条目。

---

### Phase 8: APISIX 路由集成

本期把 ParkingLotService 暴露到 APISIX，外部客户端可通过 HTTP + `grpc-transcode` 调用。所有 parking 路由都必须是 protected route，启用 `jwt-auth` 并注入身份 header：

```json
"proxy-rewrite": {
  "headers": {
    "set": {
      "x-user-id": "$jwt_payload.user_id",
      "x-tenant-id": "$jwt_payload.tenant_id",
      "x-user-role": "$jwt_payload.role"
    }
  }
}
```

**8.1 更新 proto descriptor**

`make proto-gen` 会同时执行 `buf generate` 和 `buf build -o configs/apisix/proto-descriptor.pb`。新增 parking proto 后，必须提交更新后的 `configs/apisix/proto-descriptor.pb`，否则 APISIX 的 `grpc-transcode` 无法识别 `parkhub.parking.v1.ParkingLotService`。

**8.2 更新 `configs/apisix/routes.yaml`**

沿用现有文件的 curl 示例风格，新增 parking protected routes：

| Route ID | Method | URI | gRPC method |
|----------|--------|-----|-------------|
| `parking-lot-create` | `POST` | `/parking/v1/lots` | `CreateParkingLot` |
| `parking-lot-get` | `GET` | `/parking/v1/lots/:id` | `GetParkingLot` |
| `parking-lot-list` | `GET` | `/parking/v1/lots` | `ListParkingLots` |
| `parking-lot-update` | `PATCH` | `/parking/v1/lots/:id` | `UpdateParkingLot` |
| `parking-lot-delete` | `DELETE` | `/parking/v1/lots/:id` | `DeleteParkingLot` |
| `parking-lot-stats` | `GET` | `/parking/v1/lots/stats` | `GetParkingLotStats` |

每个 route 的 `grpc-transcode` 配置使用：

```json
{
  "proto_id": "proto-descriptor",
  "service": "parkhub.parking.v1.ParkingLotService",
  "method": "<MethodName>"
}
```

注意路由匹配顺序：`/parking/v1/lots/stats` 必须优先于 `/parking/v1/lots/:id`，避免 `stats` 被当作 `id`。

**8.3 APISIX 验证**

在 `make docker-up` 后，通过 Admin API 应用新增 routes，并用 HTTP 请求验证：

```bash
# 创建
curl -X POST http://localhost:9080/parking/v1/lots \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"朝阳地下停车场","address":"朝阳区xxx","total_spaces":200,"lot_type":"LOT_TYPE_UNDERGROUND"}'

# 列表
curl http://localhost:9080/parking/v1/lots?page=1&page_size=10 \
  -H "Authorization: Bearer <access-token>"

# 详情
curl http://localhost:9080/parking/v1/lots/<lot-id> \
  -H "Authorization: Bearer <access-token>"

# 统计
curl http://localhost:9080/parking/v1/lots/stats \
  -H "Authorization: Bearer <access-token>"

# 更新
curl -X PATCH http://localhost:9080/parking/v1/lots/<lot-id> \
  -H "Authorization: Bearer <access-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"朝阳地下停车场(新)"}'

# 删除
curl -X DELETE http://localhost:9080/parking/v1/lots/<lot-id> \
  -H "Authorization: Bearer <access-token>"
```

验收点：
- 无 token 返回 401
- token 中缺少 `tenant_id` 时，backend 返回 `Unauthenticated`
- tenant-A 创建的数据，tenant-B 通过 APISIX 查询返回 NotFound
- `configs/apisix/proto-descriptor.pb` 包含 parking service，APISIX transcode 不报 unknown service/method

---

## 验证

```bash
make proto-lint && make proto-gen
go generate ./internal/domains/parking/...
go test ./internal/domains/parking/...
go test ./internal/config ./cmd/monolith ./internal/middleware
go test ./...
make build-monolith
make docker-up
./bin/parkhub &

# 1. 创建停车场（直连 gRPC，需要带 x-tenant-id 和 x-user-id metadata）
grpcurl -plaintext -H "x-tenant-id: <tenant-id>" -H "x-user-id: <user-id>" \
  -d '{"name":"朝阳地下停车场","address":"朝阳区xxx","total_spaces":200,"lot_type":"LOT_TYPE_UNDERGROUND"}' \
  localhost:50051 parkhub.parking.v1.ParkingLotService/CreateParkingLot

# 2. 获取详情
grpcurl -plaintext -H "x-tenant-id: <tenant-id>" -H "x-user-id: <user-id>" \
  -d '{"id":"<lot-id>"}' \
  localhost:50051 parkhub.parking.v1.ParkingLotService/GetParkingLot

# 3. 列表（带过滤）
grpcurl -plaintext -H "x-tenant-id: <tenant-id>" -H "x-user-id: <user-id>" \
  -d '{"status":"PARKING_LOT_STATUS_ACTIVE","pagination":{"page":1,"page_size":10}}' \
  localhost:50051 parkhub.parking.v1.ParkingLotService/ListParkingLots

# 4. 更新
grpcurl -plaintext -H "x-tenant-id: <tenant-id>" -H "x-user-id: <user-id>" \
  -d '{"id":"<lot-id>","name":"朝阳地下停车场(新)"}' \
  localhost:50051 parkhub.parking.v1.ParkingLotService/UpdateParkingLot

# 5. 统计
grpcurl -plaintext -H "x-tenant-id: <tenant-id>" -H "x-user-id: <user-id>" \
  -d '{}' \
  localhost:50051 parkhub.parking.v1.ParkingLotService/GetParkingLotStats

# 6. 删除
grpcurl -plaintext -H "x-tenant-id: <tenant-id>" -H "x-user-id: <user-id>" \
  -d '{"id":"<lot-id>"}' \
  localhost:50051 parkhub.parking.v1.ParkingLotService/DeleteParkingLot

# 7. 租户隔离验证
# 用 tenant-A 创建的车场，用 tenant-B 的 tenant-id 查询应返回 NotFound

# 8. APISIX HTTP 验证
# 按 Phase 8 应用 parking routes 后，通过 localhost:9080/parking/v1/lots* 验证 protected HTTP routes
```

---

## 文件清单

**新增 (12 个手动文件)**:

| 文件 | 说明 |
|------|------|
| `api/proto/parking/v1/parking_lot.proto` | Protobuf 定义 |
| `internal/domains/parking/domain/parking_lot.go` | 领域模型 + 行为方法 |
| `internal/domains/parking/errs/errors.go` | 哨兵错误 |
| `internal/domains/parking/repository/dao/parking_lot.go` | GORM DAO 接口 + 实现 |
| `internal/domains/parking/repository/interface.go` | Repo 接口 |
| `internal/domains/parking/repository/parking_lot_repo.go` | Repo 实现 + toDomain/toEntity |
| `internal/domains/parking/service/interface.go` | Service 接口 + DTO |
| `internal/domains/parking/service/parking_lot_service.go` | Service 实现 |
| `internal/domains/parking/grpc/parking_lot_server.go` | gRPC Handler |
| `internal/domains/parking/grpc/helpers.go` | Proto<->Domain 转换函数 |
| `internal/domains/parking/grpc/register.go` | DI 链 + 服务注册 |

**生成文件**:
| 文件 | 说明 |
|------|------|
| `internal/gen/api/proto/parking/v1/*.go` | buf 生成 |
| `internal/domains/parking/repository/dao/mocks/*.go` | mockgen 生成 |
| `internal/domains/parking/repository/mocks/*.go` | mockgen 生成 |
| `internal/domains/parking/service/mocks/*.go` | mockgen 生成 |

**测试文件**:
| 文件 | 覆盖 |
|------|------|
| `internal/domains/parking/domain/parking_lot_test.go` | 创建默认值、状态转换 |
| `internal/domains/parking/repository/dao/parking_lot_test.go` | duplicate name、租户隔离、分页过滤、统计聚合 |
| `internal/domains/parking/repository/parking_lot_repo_test.go` | DAO/domain 转换、错误透传 |
| `internal/domains/parking/service/parking_lot_service_test.go` | CRUD、分页默认值、容量不变量、GetStats |
| `internal/domains/parking/grpc/parking_lot_server_test.go` | tenantID 缺失、输入校验、错误映射、proto/domain 转换 |
| `internal/config/config_test.go` | `parking_database` 默认值、YAML、环境变量覆盖 |

**修改 (5 个文件)**:

| 文件 | 变更 |
|------|------|
| `cmd/monolith/main.go` | import parking 包，新增 parking DB 实例，AutoMigrate，RegisterServices |
| `internal/config/config.go` | 新增 `ParkingDatabase DatabaseConfig` |
| `configs/config.yaml` | 新增 parking 数据库连接配置 |
| `configs/apisix/routes.yaml` | 新增 ParkingLotService protected route 示例 |
| `configs/apisix/proto-descriptor.pb` | proto descriptor 更新，供 APISIX grpc-transcode 使用 |

**无需修改**:
- `internal/middleware/auth_context.go` — parking RPC 全部需要认证，无白名单
- `pkg/grpcutil/errors.go` — 复用现有 `ToGRPCError`

---

## 非目标

本期不做：
- Zone / Space 子实体
- 出入场事件 (Event)
- IoT 域集成（TotalGates 对接、AvailableSpaces 自动更新）
- Kafka 事件发布
- 租户车场数量同步到 identity 域的 `parking_lot_count` 字段
