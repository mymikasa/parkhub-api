# IoT 设备管理后端接入计划

## 概述

基于用户提供的 Domain + Service 定义和前端 MSW mock 数据，遵循 parking 域的分层架构实现 IoT 域。

### 用户提供的定义

**Domain 实体：**
- Device: ID(序列号), TenantID, Name, Status, FirmwareVersion, LastHeartbeat, ParkingLotID(nullable), GateID(nullable)
- DeviceStatus: `pending` / `active` / `offline` / `disabled`

**DeviceService 接口（15 个方法）：**
- CRUD: `Create`, `GetByID`, `List`, `UpdateName`, `Delete`
- 生命周期: `Bind`, `Unbind`, `Disable`, `Enable`
- 批量: `BatchDisable`, `BatchEnable`, `BatchDelete`, `BatchBind`
- 统计: `GetStats`

---

## 数据模型

### 实体关系

```
Tenant (1) ──< Device (N)
ParkingLot (1) ──< Device (N)    (nullable, 通过 Bind 赋值)
Gate (1) ──> Device (N)           (nullable, 通过 Bind 赋值)
```

### Device DAO 表结构

| 字段 | Go 类型 | DB 类型 | 说明 |
|------|---------|---------|------|
| id | string | varchar(64) | PK, 设备序列号 |
| tenant_id | string | varchar(36) | 租户 ID |
| name | string | varchar(100) | 设备名称 |
| status | string | varchar(20) | pending/active/offline/disabled |
| firmware_version | string | varchar(50) | 固件版本 |
| last_heartbeat_at | *int64 | bigint, nullable | 最后心跳时间戳(ms) |
| parking_lot_id | *string | varchar(36), nullable | 绑定停车场 ID |
| gate_id | *string | varchar(36), nullable | 绑定出入口 ID |
| created_at | int64 | bigint | 创建时间(ms) |
| updated_at | int64 | bigint | 更新时间(ms) |

---

## Service 层 DTO 定义

### Request DTOs

```go
type CreateDeviceRequest struct {
    TenantID        string
    ID              string   // 设备序列号
    Name            string
    FirmwareVersion string
}

type GetDeviceRequest struct {
    TenantID string
    ID       string
}

type ListDevicesRequest struct {
    TenantID     string
    Status       domain.DeviceStatus  // 可选过滤
    ParkingLotID string               // 可选过滤
    Keyword      string               // 可选，按 name/ID 搜索
    Page         int
    PageSize     int
}

type UpdateDeviceNameRequest struct {
    TenantID string
    ID       string
    Name     string
}

type BindDeviceRequest struct {
    TenantID     string
    ID           string
    ParkingLotID string
    GateID       string
}

type UnbindDeviceRequest struct {
    TenantID string
    ID       string
}

type ChangeDeviceStatusRequest struct {
    TenantID string
    ID       string
}

type DeleteDeviceRequest struct {
    TenantID string
    ID       string
}

type BatchChangeDeviceStatusRequest struct {
    TenantID string
    IDs      []string
}

type BatchDeleteDeviceRequest struct {
    TenantID string
    IDs      []string
}

type BatchBindDeviceRequest struct {
    TenantID string
    Bindings []struct {
        ID           string
        ParkingLotID string
        GateID       string
    }
}
```

### Response DTOs

```go
type DeviceListResponse struct {
    Devices    []*domain.Device
    Total      int64
    Page       int
    PageSize   int
    TotalPages int
}

type DeviceStatsResponse struct {
    Total    int64
    Active   int64
    Offline  int64
    Pending  int64
    Disabled int64
}
```

---

## Proto 定义

### 包名: `parkhub.iot.v1`

### DeviceService

```protobuf
service DeviceService {
  rpc CreateDevice(CreateDeviceRequest) returns (CreateDeviceResponse);
  rpc GetDevice(GetDeviceRequest) returns (GetDeviceResponse);
  rpc ListDevices(ListDevicesRequest) returns (ListDevicesResponse);
  rpc UpdateDeviceName(UpdateDeviceNameRequest) returns (UpdateDeviceNameResponse);
  rpc BindDevice(BindDeviceRequest) returns (BindDeviceResponse);
  rpc UnbindDevice(UnbindDeviceRequest) returns (UnbindDeviceResponse);
  rpc DisableDevice(DisableDeviceRequest) returns (DisableDeviceResponse);
  rpc EnableDevice(EnableDeviceRequest) returns (EnableDeviceResponse);
  rpc DeleteDevice(DeleteDeviceRequest) returns (DeleteDeviceResponse);
  rpc BatchDisableDevices(BatchChangeDeviceStatusRequest) returns (BatchChangeDeviceStatusResponse);
  rpc BatchEnableDevices(BatchChangeDeviceStatusRequest) returns (BatchChangeDeviceStatusResponse);
  rpc BatchDeleteDevices(BatchDeleteDeviceRequest) returns (BatchDeleteDevicesResponse);
  rpc BatchBindDevices(BatchBindDeviceRequest) returns (BatchBindDevicesResponse);
  rpc GetDeviceStats(GetDeviceStatsRequest) returns (GetDeviceStatsResponse);
}
```

### 枚举

```protobuf
enum DeviceStatus {
  DEVICE_STATUS_UNSPECIFIED = 0;
  DEVICE_STATUS_PENDING = 1;
  DEVICE_STATUS_ACTIVE = 2;
  DEVICE_STATUS_OFFLINE = 3;
  DEVICE_STATUS_DISABLED = 4;
}
```

---

## 前后端 API 映射

### APISIX 路由

| Route | Method | URI | → gRPC |
|-------|--------|-----|--------|
| 60 | POST | `/iot/v1/devices` | `CreateDevice` |
| 61 | GET | `/iot/v1/devices/:id` | `GetDevice` |
| 62 | GET | `/iot/v1/devices` | `ListDevices` |
| 63 | PATCH | `/iot/v1/devices/:id/name` | `UpdateDeviceName` |
| 64 | POST | `/iot/v1/devices/:id/bind` | `BindDevice` |
| 65 | POST | `/iot/v1/devices/:id/unbind` | `UnbindDevice` |
| 66 | POST | `/iot/v1/devices/:id/disable` | `DisableDevice` |
| 67 | POST | `/iot/v1/devices/:id/enable` | `EnableDevice` |
| 68 | DELETE | `/iot/v1/devices/:id` | `DeleteDevice` |
| 69 | POST | `/iot/v1/devices/batch/disable` | `BatchDisableDevices` |
| 70 | POST | `/iot/v1/devices/batch/enable` | `BatchEnableDevices` |
| 71 | POST | `/iot/v1/devices/batch/delete` | `BatchDeleteDevices` |
| 72 | POST | `/iot/v1/devices/batch/bind` | `BatchBindDevices` |
| 73 | GET | `/iot/v1/devices/stats` | `GetDeviceStats` |

### 前端字段映射

| 前端 (camelCase) | 后端 (snake_case/proto) | 转换规则 |
|---|---|---|
| `serialNumber` | `id` | 前端 serialNumber = 后端 Device.ID |
| `status: "online"` | `status: DEVICE_STATUS_ACTIVE` | 前端 online ↔ active |
| `status: "offline"` | `status: DEVICE_STATUS_OFFLINE` | 直接映射 |
| `parkingLotId` | `parking_lot_id` | camelCase → snake_case |
| `lastHeartbeat` | `last_heartbeat_at` | camelCase → snake_case + timestamp |
| `type: "integrated"` etc. | (暂无对应字段) | 前端有 type，后端暂无 |

---

## 分层实现计划

### Phase 0: 基础设施

**修改文件 (4):**

| 文件 | 改动 |
|---|---|
| `internal/config/config.go` | 添加 `IoTDatabase DatabaseConfig` + 默认值 + `IOT_DB_*` env |
| `configs/config.yaml` | 添加 `iot_database:` 段 |
| `docker-compose.yml` | 添加 `IOT_DB_*` 环境变量到 monolith |
| `cmd/monolith/main.go` | IoT DB 连接、AutoMigrate、注册 gRPC |

### Phase 1: Domain + Errors

**新建文件 (3):**

| 文件 | 说明 |
|---|---|
| `internal/domains/iot/errs/errors.go` | 6 个领域错误 |
| `internal/domains/iot/domain/device.go` | Device 实体 + 状态机 (pending→active→offline↔disabled) + Bind/Unbind 业务方法 |
| `internal/domains/iot/domain/device_test.go` | 状态转换、Bind/Unbind 规则测试 |

### Phase 2: DAO (TDD)

**新建文件 (4):**

| 文件 | 说明 |
|---|---|
| `internal/domains/iot/repository/dao/device.go` | GORM DAO 接口 + 实现 |
| `internal/domains/iot/repository/dao/device_test.go` | SQLite 集成测试 |
| `internal/domains/iot/repository/dao/mocks/device.mock.go` | mockgen |

**DAO 方法:**
- `Insert(ctx, *Device) error`
- `FindByID(ctx, id string) (*Device, error)`
- `FindAll(ctx, filter DeviceFilter, page, pageSize int) ([]*Device, int64, error)`
- `Update(ctx, *Device) error` — 使用 `Select()` 持久化零值
- `Delete(ctx, id string) error`
- `DeleteBatch(ctx, ids []string) error`
- `CountByStatus(ctx, tenantID string) (pending, active, offline, disabled int64, err error)`
- `FindByParkingLotID(ctx, parkingLotID string) ([]*Device, error)`

### Phase 3: Repository (TDD)

**新建文件 (3):**

| 文件 | 说明 |
|---|---|
| `internal/domains/iot/repository/interface.go` | DeviceRepo 接口 |
| `internal/domains/iot/repository/device_repo.go` | DAO → Domain 适配 |
| `internal/domains/iot/repository/device_repo_test.go` | |

### Phase 4: Service (TDD)

**新建文件 (4):**

| 文件 | 说明 |
|---|---|
| `internal/domains/iot/service/interface.go` | DeviceService 接口 + 所有 Request/Response DTO |
| `internal/domains/iot/service/device_service.go` | 15 个方法实现 |
| `internal/domains/iot/service/device_service_test.go` | |
| `internal/domains/iot/service/mocks/device_service.mock.go` | mockgen |

### Phase 5: gRPC + Proto (TDD)

**新建文件 (6):**

| 文件 | 说明 |
|---|---|
| `api/proto/iot/v1/device.proto` | DeviceService + 枚举 + 所有 messages |
| `internal/gen/api/proto/iot/v1/*` | `make proto-gen` |
| `internal/domains/iot/grpc/device_server.go` | 14 个 RPC handler |
| `internal/domains/iot/grpc/device_server_test.go` | |
| `internal/domains/iot/grpc/helpers.go` | Proto ↔ Domain 转换 |
| `internal/domains/iot/grpc/register.go` | 注册到 Registry |

### Phase 6: 路由 + 文档

**修改文件 (2):**

| 文件 | 改动 |
|---|---|
| `scripts/init-apisix.sh` | 添加 routes 60-73 |
| `configs/swagger/openapi.yaml` | Device schemas + paths |

### Phase 7: 前端适配

**修改文件 (2):**

| 文件 | 改动 |
|---|---|
| `parkhub-web/src/lib/api/client.ts` | 添加 `/iot/` 到 REAL_API_PREFIXES |
| `parkhub-web/src/lib/api/devices.ts` | URL 改为 `/iot/v1/devices/*`，适配新响应格式 |

---

## 待确认事项

| # | 问题 | 影响 |
|---|------|------|
| 1 | Gate（出入口）是否属于 iot 域还是 parking 域？ | 决定 Gate DAO 放哪个域 |
| 2 | Device 是否需要 `type` 字段（一体机/仅摄像头/仅道闸）？前端有，domain 没给 | Proto + DAO 表结构 |
| 3 | `todayTraffic` 如何计算？Device 表加字段还是后续接入 Kafka？ | GetStats 返回值 |
| 4 | BatchBind 的 bindings 结构确认？ | Proto message 定义 |
| 5 | 前端 `SendDeviceCommand`（抬杆/落杆）是否在后端实现？Service 没有这个方法 | 是否需要额外 RPC |
| 6 | Device.ID 是用户传入的序列号还是后端生成的 UUID？ | CreateDevice 逻辑 |

---

## 文件清单

### 新建 (~20 个文件)
```
api/proto/iot/v1/device.proto
internal/domains/iot/errs/errors.go
internal/domains/iot/domain/device.go
internal/domains/iot/domain/device_test.go
internal/domains/iot/repository/dao/device.go
internal/domains/iot/repository/dao/device_test.go
internal/domains/iot/repository/dao/mocks/device.mock.go
internal/domains/iot/repository/interface.go
internal/domains/iot/repository/device_repo.go
internal/domains/iot/repository/device_repo_test.go
internal/domains/iot/repository/mocks/device_repo.mock.go
internal/domains/iot/service/interface.go
internal/domains/iot/service/device_service.go
internal/domains/iot/service/device_service_test.go
internal/domains/iot/service/mocks/device_service.mock.go
internal/domains/iot/grpc/device_server.go
internal/domains/iot/grpc/device_server_test.go
internal/domains/iot/grpc/helpers.go
internal/domains/iot/grpc/register.go
```

### 修改 (6 个后端 + 2 个前端)
```
internal/config/config.go
configs/config.yaml
docker-compose.yml
cmd/monolith/main.go
scripts/init-apisix.sh
configs/swagger/openapi.yaml
---
parkhub-web/src/lib/api/client.ts
parkhub-web/src/lib/api/devices.ts
```
