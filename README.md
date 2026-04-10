# parkhub-api

微服务架构的 ParkHub SaaS 停车管理系统后端

## 1. 架构概览

采用 **Monolith-First** 策略：初期以单进程 monolith 运行，domain 间通过 in-process 调用；待业务验证后按 domain 拆分为独立微服务，通过 K8s Service 通信。

### 1.1 Domain 划分

| Domain | Database | 职责 |
|--------|----------|------|
| identity | parkhub_identity | 认证、用户、租户、角色 |
| parking | parkhub_parking | 车场、车位、区域、出入场事件 |
| iot | parkhub_iot | 摄像头、道闸、传感器设备管理 |
| billing | parkhub_billing | 计费规则、费用计算 |
| order | parkhub_order | 停车订单、费用明细、订单状态 |
| payment | parkhub_payment | 支付渠道集成、支付记录 |

### 1.2 关键技术选型

| 领域 | 选型 | 备注 |
|------|------|------|
| 语言 | Go 1.26+ | |
| 服务间通信 | gRPC + Protobuf | 单进程阶段为 in-process，多进程阶段为 K8s Service |
| 前后端通信 | gRPC-Gateway (`grpc-ecosystem/grpc-gateway/v2`) | proto 中定义 HTTP 映射，自动生成 RESTful 反向代理 |
| 消息总线 | Kafka (KRaft 模式) | 出入场事件回溯/重放/对账 |
| API 网关 | APISIX | 热更新、etcd 存储、JWT 验签 |
| 数据库 | MySQL 8.0 + 分 Database | 每个 domain 独立 database（6 个），物理上共享实例 |
| ORM | GORM (`gorm.io/driver/mysql`) | |
| 数据库迁移 | goose (`github.com/pressly/goose/v3`) | 支持 `embed.FS` 单二进制部署 |
| 缓存 | Redis (`github.com/redis/go-redis/v9`) | 车位计数、限流、计费规则缓存、JWT 黑名单 |
| 配置管理 | Viper (`github.com/spf13/viper`) | YAML + 环境变量覆盖 |
| 日志 | `log/slog` | `JSONHandler` (prod) / `TextHandler` (dev) |
| 认证 | JWT RS256 (`github.com/golang-jwt/jwt/v5`) | APISIX 网关验签，注入租户/用户头 |
| 授权 | RBAC 中间件 | 基于 `OperatorRole` 枚举的权限矩阵 |
| 依赖注入 | Google Wire (`github.com/google/wire`) | 编译期代码生成 |
| HTTP 路由 | gRPC-Gateway `runtime.ServeMux` | 健康检查等非 RPC 路由通过自定义 handler 挂载 |
| 参数校验 | protovalidate (`github.com/bufbuild/protovalidate-go`) | CEL 表达式，规则写在 proto 中 |
| 代码检查 | golangci-lint v2 | + 自定义 tenant linter |
| 测试 | testify + mockery + testcontainers-go | |
| 租户隔离 | Pool 默认 + Bridge 按需 + Silo 私有化 | ORM Callback 强制 `tenant_id` 注入 |
| 时序数据 | MySQL 原生分区表（按月 `PARTITION BY RANGE`） | |
| 规则存储 | MySQL JSON 列 + 虚拟生成列索引 | |
| 容器编排 | Kubernetes | |
| 指标采集 | VictoriaMetrics + vmagent | PromQL 兼容 |
| 链路追踪 | OpenTelemetry + Tempo | |
| 日志聚合 | Loki | |
| 可视化 | Grafana | |

### 1.3 错误处理

- 每个 domain 定义领域错误（`var ErrXxxNotFound = errors.New(...)`）
- `pkg/grpcutil/errors.go` 将领域错误映射到 gRPC 状态码
- gRPC-Gateway 自动将 gRPC 状态码转换为对应 HTTP 状态码

### 1.4 熔断/韧性

- Monolith 阶段暂不引入（进程内调用无意义）
- 拆分后使用 `github.com/sony/gobreaker`，作为 gRPC interceptor 实现

## 2. 项目结构

```
cmd/
  monolith/                # 单体入口
    main.go, wire.go, wire_gen.go
internal/
  config/                  # Viper 配置结构体
  data/                    # 多数据库初始化、连接管理
  middleware/
    tenant.go              # GORM tenant_id 注入 Callback
    auth.go                # gRPC interceptor（JWT claims → context）
    logging.go             # slog 请求日志 gRPC interceptor
  domains/
    identity/              # 认证、用户、租户
    parking/               # 车场、车位、区域、出入场事件
    iot/                   # 摄像头、道闸、传感器
    billing/               # 计费规则、费用计算
    order/                 # 停车订单、费用明细、订单状态
    payment/               # 支付渠道集成
  gen/                     # buf 生成代码
  registry/                # 服务注册（gRPC Server）
pkg/
  grpcutil/                # gRPC 错误映射
  auth/                    # JWT 解析、RBAC 矩阵
migrations/
  parkhub_identity/
  parkhub_parking/
  parkhub_iot/
  parkhub_billing/
  parkhub_order/
  parkhub_payment/
tools/
  tenantlint/              # 自定义 go vet 分析器
```

## 3. 开发指南

### 3.1 本地环境

```bash
# 启动基础设施
make docker-up

# 运行数据库迁移
make migrate

# 生成 proto 代码
make proto-gen

# 生成 Wire 依赖注入代码
make wire

# 构建并运行
make build-monolith && ./bin/parkhub
```

### 3.2 Makefile 目标

| 目标 | 说明 |
|------|------|
| `proto-gen` | buf generate |
| `proto-lint` | buf lint |
| `proto-breaking` | buf breaking 检查 |
| `lint` | golangci-lint run |
| `lint-tenant` | 自定义租户 linter |
| `test` | go test ./... |
| `test-integration` | go test -tags=integration ./... |
| `build-monolith` | go build -o bin/parkhub ./cmd/monolith |
| `wire` | wire ./cmd/monolith/... |
| `migrate` | goose 迁移 |
| `docker-up` | docker compose up -d |
| `docker-down` | docker compose down |

### 3.3 Docker Compose 服务并启动

| 服务 | 镜像 | 端口 |
|------|------|------|
| mysql | mysql:8.0 | 3306 |
| redis | redis:7-alpine | 6379 |
| kafka | bitnamilegacy/kafka:4.0.0-debian-12-r10 (KRaft) | 9092 |
| apisix | apache/apisix:3.15.0-debian | 9080 |
| etcd | bitnamilegacy/etcd:3.6.4-debian-12-r4 | 2379 |
| grafana | grafana/grafana | 3000 |
| victoriametrics | victoriametrics/victoria-metrics | 8428 |
| tempo | grafana/tempo:2.3.1 | 3200, 4317, 4318 |
| loki | grafana/loki | 3100 |
| grafana | grafana/grafana | 3000 |

使用仓库根目录下的 [`docker-compose.yml`](/Users/mikasa/mikasa/parkhub-api/docker-compose.yml) 启动本地基础设施：

```bash
# 方式一：直接使用 Docker Compose
docker compose up -d

# 方式二：使用 Makefile 包装命令
make docker-up
```

启动后可通过以下命令确认服务状态：

```bash
docker compose ps
docker compose logs -f mysql redis kafka etcd apisix
```

其中，部分基础设施使用仓库内显式配置文件：

- APISIX: [`configs/apisix/config.yaml`](/Users/mikasa/mikasa/parkhub-api/configs/apisix/config.yaml)
- Tempo: [`configs/tempo/tempo.yaml`](/Users/mikasa/mikasa/parkhub-api/configs/tempo/tempo.yaml)

常用访问入口：

| 服务 | 地址 | 默认凭据 |
|------|------|----------|
| MySQL | `127.0.0.1:3306` | `root/parkhub` 或 `parkhub/parkhub` |
| Redis | `127.0.0.1:6379` | 无 |
| Kafka | `127.0.0.1:9092` | 无 |
| etcd | `http://127.0.0.1:2379` | 无 |
| APISIX | `http://127.0.0.1:9080` | 无 |
| VictoriaMetrics | `http://127.0.0.1:8428` | 无 |
| Tempo | `http://127.0.0.1:3200` | 无 |
| Loki | `http://127.0.0.1:3100` | 无 |
| Grafana | `http://127.0.0.1:3000` | `admin/admin` |

停止并清理容器：

```bash
docker compose down
# 或
make docker-down
```
