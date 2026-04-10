# CLAUDE.md

ParkHub SaaS 停车管理系统后端。采用 Monolith-First 策略，单进程运行 6 个业务域，后期按域拆分为独立微服务。

**技术栈**: Go 1.26.1 · gRPC + Protobuf (buf) · GORM · Kafka · APISIX · Redis

## 常用命令

```bash
make docker-up          # 启动基础设施 (MySQL, Redis, Kafka, etcd, APISIX, Grafana 等)
make docker-down        # 停止基础设施
make proto-gen          # buf generate 生成 proto 代码到 internal/gen/
make proto-lint         # buf lint 检查 proto 规范
make build-monolith     # go build -o bin/parkhub ./cmd/monolith
make test               # go test ./...
make lint               # golangci-lint run
make lint-tenant        # 自定义租户 linter（确保 tenant_id 隔离）
make wire               # Google Wire 依赖注入代码生成
make migrate            # goose 数据库迁移
```

## 项目结构

```
cmd/monolith/main.go           # 单体入口，初始化 DB + gRPC Server
internal/
  domains/                      # 业务域（identity, parking, iot, billing, order, payment）
  gen/                          # buf 生成的 Go/gRPC 代码（勿手动编辑）
  middleware/                   # gRPC interceptor: tenant_id 注入、JWT auth、logging
  registry/registry.go          # gRPC 服务注册管理器
api/proto/                      # Protobuf 定义，按 parkhub.<domain>.v1 组织
pkg/grpcutil/errors.go          # 领域错误 → gRPC 状态码映射
configs/                        # 应用配置 + APISIX + Tempo
scripts/init-databases.sql      # Docker 初始化 6 个域数据库
```

## 代码规范

- **Proto 包名**: `parkhub.<domain>.v1`，Go option 输出到 `internal/gen/`
- **领域错误**: 每个 domain 定义 `var ErrXxx = errors.New(...)`，通过 `pkg/grpcutil.ToGRPCError()` 映射
- **参数校验**: protovalidate + CEL 表达式，规则写在 proto 文件中
- **租户隔离**: GORM Callback 强制注入 `tenant_id`，自定义 linter 检查遗漏
- **依赖注入**: Google Wire 编译期生成

## 基础设施 (Docker Compose)

MySQL 8.0 (:3306) · Redis (:6379) · Kafka KRaft (:9092) · etcd (:2379) · APISIX (:9080) · VictoriaMetrics (:8428) · Tempo (:3200) · Loki (:3100) · Grafana (:3000, admin/admin)

## 域划分

| 域 | 数据库 | 职责 |
|---|--------|------|
| identity | parkhub_identity | 认证、用户、租户、角色 |
| parking | parkhub_parking | 车场、车位、区域、出入场事件 |
| iot | parkhub_iot | 摄像头、道闸、传感器设备管理 |
| billing | parkhub_billing | 计费规则、费用计算 |
| order | parkhub_order | 停车订单、费用明细、订单状态 |
| payment | parkhub_payment | 支付渠道集成、支付记录 |
