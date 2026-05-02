# ParkHub 开发流程

## 1. Proto 定义

```
api/proto/<domain>/v1/<entity>.proto
```

- 包名：`parkhub.<domain>.v1`
- 复用 `common/v1/pagination.proto`、`common/v1/enums.proto`
- 校验规则用 `buf validate`（CEL 表达式写在 proto 中）

```bash
make proto-lint      # 规范检查
make proto-gen       # 生成 Go 代码 + proto descriptor
```

## 2. 领域实现

```
internal/domains/<domain>/
├── model/            # 领域模型（纯 Go struct）
├── repository/
│   ├── interface.go  # Repository 接口定义
│   └── dao/          # GORM 模型 + 实现
├── service/          # 业务逻辑
│   ├── xxx_service.go
│   ├── xxx_service_test.go
│   └── mocks/        # gomock 生成
├── grpc/             # gRPC server
│   ├── xxx_server.go
│   ├── xxx_server_test.go
│   └── errors.go     # 领域错误 → gRPC 状态码
```

- 领域错误：`var ErrXxx = errors.New(...)` → `pkg/grpcutil.ToGRPCError()`
- 租户隔离：GORM Callback 自动注入 `tenant_id`，`make lint-tenant` 检查遗漏
- 依赖注入：Google Wire

## 3. 注册服务

`internal/domains/<domain>/grpc/` 提供 `RegisterServices(*registry.Registry, *gorm.DB)`，在 `cmd/monolith/main.go` 中调用。

## 4. 数据库

```bash
# 迁移文件
migrations/<domain>/NNNN_create_xxx.up.sql

# 初始化脚本添加数据库
scripts/init-databases.sql  → CREATE DATABASE parkhub_<domain>;

make migrate
```

## 5. 测试

```bash
make test              # 单元测试
make lint              # golangci-lint
make lint-tenant       # 租户隔离检查
make build-monolith    # 编译验证
```

## 6. 暴露 HTTP API

在 `scripts/init-apisix.sh` 中添加路由：

```bash
# 1. make proto-gen 已自动生成 proto-descriptor.pb
# 2. 添加路由（照 TenantService 模式）
#    - POST/GET 无路径参数：直接 grpc-transcode
#    - GET/DELETE 带路径参数：serverless-pre-function 注入 query args
#    - PUT 带路径参数：serverless-pre-function 注入 request body
#    - 所有路由加 opentelemetry + prometheus 插件
```

## 7. Postman Collection

在 `configs/postman/parkhub-api.json` 中添加对应的请求。

## 8. 部署验证

```bash
make docker-up         # 启动所有基础设施
curl http://localhost:9080/api/v1/<resource>  # 验证 API
```

### 可观测性验证

- **Metrics**：`http://localhost:3000`（Grafana，数据源 VictoriaMetrics）
- **Traces**：`http://localhost:3000`（数据源 Tempo），确认 APISIX → monolith trace 串联
- **Logs**：`http://localhost:3000`（数据源 Loki）
