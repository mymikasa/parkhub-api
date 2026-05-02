# 用户管理 产品需求文档 (PRD)

**创建时间**: 2026-04-21
**状态**: Draft
**优先级**: P0

---

## 1. 相关用户故事

> 详细故事与验收标准请查看 `docs/user-stories/` 中对应文档。

### 1.1 相关故事

- `[US-AUTH-001]` 创建租户用户账号，优先级 P0，来源 `docs/user-stories/01-platform-admin-user-stories.md`
  - 角色：平台管理员
  - 摘要：平台管理员为租户创建管理员或操作员账号，指定角色与租户归属

- `[US-AUTH-002]` 批量导入用户，优先级 P1，来源 `docs/user-stories/01-platform-admin-user-stories.md`
  - 角色：平台管理员
  - 摘要：一次性批量导入多个用户账号，支持部分失败并返回错误明细

- `[US-AUTH-003]` 查看用户列表，优先级 P0，来源 `docs/user-stories/01-platform-admin-user-stories.md`
  - 角色：平台管理员
  - 摘要：按租户、角色、状态、关键词筛选和分页浏览用户列表

- `[US-AUTH-004]` 冻结和解冻用户，优先级 P0，来源 `docs/user-stories/01-platform-admin-user-stories.md`
  - 角色：平台管理员
  - 摘要：冻结异常账号或解冻恢复正常账号，控制登录访问

- `[US-AUTH-005]` 重置用户密码，优先级 P0，来源 `docs/user-stories/01-platform-admin-user-stories.md`
  - 角色：平台管理员
  - 摘要：管理员为用户重置密码，无需知道旧密码

- `[US-AUTH-006]` 编辑用户信息，优先级 P1，来源 `docs/user-stories/01-platform-admin-user-stories.md`
  - 角色：平台管理员
  - 摘要：修改用户的用户名、邮箱、手机号、真实姓名或角色

- `[US-AUTH-007]` 创建租户内操作员账号，优先级 P0，来源 `docs/user-stories/02-tenant-admin-user-stories.md`
  - 角色：租户管理员
  - 摘要：租户管理员在本租户范围内创建操作员账号

- `[US-AUTH-008]` 查看本租户用户列表，优先级 P0，来源 `docs/user-stories/02-tenant-admin-user-stories.md`
  - 角色：租户管理员
  - 摘要：查看仅属于本租户的用户，不暴露跨租户数据

- `[US-AUTH-009]` 冻结本租户用户，优先级 P1，来源 `docs/user-stories/02-tenant-admin-user-stories.md`
  - 角色：租户管理员
  - 摘要：冻结或解冻本租户内的操作员账号

- `[US-AUTH-010]` 查看和修改个人资料，优先级 P1，来源 `docs/user-stories/03-operator-user-stories.md`
  - 角色：操作员
  - 摘要：用户更新自己的真实姓名、邮箱、手机号

- `[US-AUTH-011]` 修改自己的密码，优先级 P0，来源 `docs/user-stories/03-operator-user-stories.md`
  - 角色：操作员
  - 摘要：通过验证旧密码后设置新密码

- `[US-AUTH-012]` 查看当前登录用户信息，优先级 P0，来源 `docs/user-stories/03-operator-user-stories.md`
  - 角色：操作员
  - 摘要：获取当前认证用户的基本信息与角色

### 1.2 优先级汇总

| 优先级 | 数量 | 关键故事 |
|--------|------|----------|
| P0 | 8 | US-AUTH-001、003、004、005、007、008、011、012 |
| P1 | 4 | US-AUTH-002、006、009、010 |
| P2 | 0 | — |

---

## 2. 范围界定

### 2.1 包含功能

- 用户账号的创建、查询、更新（用户名、邮箱、手机号、真实姓名、角色）
- 用户状态管理：激活、冻结、解冻
- 管理员重置用户密码
- 用户自助修改密码（需验证旧密码）
- 用户自助更新个人资料（真实姓名、邮箱、手机号）
- 获取当前登录用户信息
- 用户列表分页查询，支持按租户、角色、状态、关键词过滤
- 批量导入用户（支持部分失败，返回失败明细）
- 角色体系：`platform_admin`、`tenant_admin`、`operator`
- 租户隔离：`tenant_admin` 只能管理本租户用户，`platform_admin` 可跨租户

### 2.2 不包含功能 (Out of Scope)

- 用户注册（自助注册流程，终端消费者场景）— 由 `auth` 域认证功能负责
- OAuth2 / SSO 第三方登录集成
- 用户权限的细粒度配置（RBAC 扩展）
- 用户操作日志审计
- 用户删除（当前不支持物理删除用户，仅支持冻结）
- 多因素认证 (MFA) / TOTP

### 2.3 依赖项

- **identity 域 - 认证模块**：用户管理依赖认证拦截器注入 `x-user-id`、`x-tenant-id`、`x-user-role`，才能在服务层执行权限判断（已实现，见 `internal/middleware/auth_context.go`）
- **租户管理模块**：创建用户时，`tenant_id` 必须对应已存在的租户（当前未做外键约束，为待补充项）
- **密码策略**：当前策略为密码长度 ≥ 8 位，未来可能扩展复杂度规则

---

## 3. 需求概述

### 3.1 功能描述

用户管理是 ParkHub SaaS 平台 identity 域的核心能力之一，负责对平台内所有人员账号进行全生命周期管理。系统支持三个角色层级（平台管理员、租户管理员、操作员），并通过租户隔离确保数据安全。

平台管理员拥有全局视角，可以跨租户创建和管理用户；租户管理员只能管理本租户范围内的用户；操作员无用户管理权限，仅能维护自己的个人信息。

### 3.2 关键特性

- **角色分层**: 三层角色体系，权限从平台管理员向下递减
- **租户隔离**: 数据访问边界严格按 `tenant_id` 隔离，`tenant_admin` 无法看到或操作其他租户的用户
- **状态机**: 用户状态仅允许合法转换（激活→冻结，冻结→激活），非法转换返回错误
- **密码安全**: 密码使用 bcrypt 加密存储，最低长度限制 8 位，修改密码需验证旧密码
- **批量操作**: 支持批量导入用户，每条记录独立处理，部分失败不影响其余记录

---

## 4. 当前实现状态

| 功能模块 | 状态 | 备注 |
|---------|------|------|
| 创建用户 | ✅ | `UserService.Create`，含用户名唯一校验和密码 bcrypt 加密 |
| 获取用户详情 | ✅ | `UserService.GetByID` |
| 获取当前用户 | ✅ | `user_server.go GetCurrentUser`，从 `identityctx` 中取 user_id |
| 用户列表查询 | ✅ | `UserService.List`，支持 tenant_id/role/status/keyword 过滤和分页 |
| 更新用户信息 | ✅ | `UserService.Update`，支持部分字段更新 |
| 冻结用户 | ✅ | `UserService.Freeze`，含状态转换校验 |
| 解冻用户 | ✅ | `UserService.Unfreeze`，含状态转换校验 |
| 重置密码（管理员） | ✅ | `UserService.ResetPassword` |
| 修改个人资料 | ✅ | `UserService.UpdateProfile`（真实姓名、邮箱、手机号） |
| 修改密码（用户自助） | ✅ | `UserService.ChangePassword`，验证旧密码 |
| 批量导入用户 | ✅ | `UserService.ImportUsers`，部分失败返回错误明细 |
| 角色权限校验（服务层） | ⚠️ | `UnaryAuthContextInterceptor` 注入身份，但服务层未全面做角色鉴权判断，需补充 |
| 租户边界隔离（服务层） | ⚠️ | `tenant_id` 注入已实现，但 `tenant_admin` 跨租户操作的拦截逻辑未完全覆盖 |
| 创建用户时 tenant_id 合法性校验 | ❌ | 当前不验证 tenant_id 是否属于已存在的租户 |

---

## 5. 功能需求

### 5.1 核心需求

**FR-1：用户创建**
- 平台管理员可创建任意角色（platform_admin、tenant_admin、operator）的用户，可指定或不指定 `tenant_id`。
- 租户管理员只能在本租户内创建 `operator` 角色用户，不能创建 `tenant_admin` 或 `platform_admin`。
- 用户名在全系统范围内唯一，重复时返回"用户名已存在"错误。
- 密码长度最低 8 位，不满足时返回"密码长度不足"错误。

**FR-2：用户查询**
- 用户列表支持按 `tenant_id`、`role`、`status`、`keyword` 过滤，支持分页（默认 20 条/页，最大 100 条）。
- `platform_admin` 可查询全局用户；`tenant_admin` 只能查询本 `tenant_id` 下的用户。
- 操作员不具备用户列表查询权限。

**FR-3：用户状态管理**
- 用户状态为有限状态机：`active` ↔ `frozen`，不合法的状态转换返回"无效状态转换"错误。
- `platform_admin` 可冻结/解冻全局用户。
- `tenant_admin` 只能冻结/解冻本租户用户。
- 被冻结的用户登录时被系统拒绝，返回"账号已被冻结"错误（由认证服务处理）。

**FR-4：密码管理**
- 管理员重置密码：无需旧密码，直接设置新密码；仅限 `platform_admin` 和对应层级的 `tenant_admin` 操作。
- 用户自助修改密码：必须先验证旧密码；旧密码错误返回"原密码不正确"。
- 新密码长度最低 8 位。

**FR-5：个人资料**
- 用户可更新自己的真实姓名、邮箱、手机号，字段独立可选（patch 语义）。
- 用户不能修改自己的角色或 `tenant_id`。

**FR-6：批量导入**
- 每条记录独立处理；单条失败不中断其他记录。
- 返回总数、成功数、失败数和失败明细（索引 + 原因）。

### 5.2 验收目标

- 平台管理员可完整执行用户的增删改查和状态管理。
- 租户管理员不能看到或操作其他租户的用户。
- 操作员只能修改自己的个人资料和密码，无法访问其他用户数据。
- 所有密码以 bcrypt 加密存储，接口中不返回明文密码或密码 hash。
- 非法状态转换（冻结→冻结、激活→激活）返回明确错误。
- 批量导入部分失败不影响整批请求，失败条目有清晰的索引和原因。

---

## 6. API 相关约束

**状态**: 必填

**访问控制原则**:
- 所有用户管理接口（除 `GetCurrentUser`）要求调用方携带有效 access token，由 APISIX 网关验证并在 gRPC metadata 中注入 `x-user-id`、`x-tenant-id`、`x-user-role`。
- `UserService` 内各 RPC 的权限边界：
  - `CreateUser`、`ListUsers`、`FreezeUser`、`UnfreezeUser`、`ResetPassword`：需 `platform_admin` 或 `tenant_admin`；`tenant_admin` 操作范围限制在自身 `tenant_id`。
  - `UpdateUser`：需 `platform_admin` 或 `tenant_admin`。
  - `GetCurrentUser`、`UpdateProfile`、`ChangePassword`：任意已认证用户均可访问，但只能操作自身数据。
  - `ImportUsers`：仅 `platform_admin`。
- 租户数据隔离：`tenant_admin` 发起的请求必须在服务层验证目标 user 的 `tenant_id` 与请求方 `tenant_id` 一致，否则返回权限错误。

**接口能力位置**:
- Proto 定义：`api/proto/identity/v1/user.proto`
- gRPC 实现：`internal/domains/identity/grpc/user_server.go`

---

## 7. 前端/交互约束

**状态**: 不适用（当前系统以 gRPC 服务为主，前端由独立团队对接；本 PRD 不规定 UI 实现）

如后续需要管理控制台前端，关键交互原则为：
- 用户列表页需支持按租户、角色、状态的组合筛选。
- 冻结/解冻操作需二次确认弹窗，防止误操作。
- 批量导入完成后显示导入结果摘要（成功 N 条，失败 M 条），失败条目可下载明细。
- 修改密码操作需隐藏密码输入（password 类型），确认新密码一致性校验在前端完成。

---

## 8. 技术设计承接

**状态**: 部分存在

当前业务逻辑已在以下文件中实现，如需了解接口细节、数据模型或服务层设计，请参阅：
- 领域模型：`internal/domains/identity/domain/user.go`
- 服务层：`internal/domains/identity/service/user_service.go`
- gRPC 层：`internal/domains/identity/grpc/user_server.go`
- 仓储层：`internal/domains/identity/repository/user_repo.go`
- 错误定义：`internal/domains/identity/errs/errors.go`

**待补充设计**：
- 服务层角色鉴权的完整实现方案（当前仅在 middleware 注入，服务层判断未全覆盖）
- 租户边界隔离的跨服务验证策略
- 如需前端接入，对应 BFF 或 OpenAPI 适配方案可由 `/t-design` 产出

---

## 9. 相关文件索引

### 9.1 后端文件

- `api/proto/identity/v1/user.proto` - UserService RPC 定义（已实现）
- `internal/domains/identity/domain/user.go` - 用户领域模型，角色和状态枚举（已实现）
- `internal/domains/identity/service/user_service.go` - 用户服务业务逻辑（已实现）
- `internal/domains/identity/service/user_interface.go` - UserService 接口定义
- `internal/domains/identity/grpc/user_server.go` - gRPC Handler 实现（已实现）
- `internal/domains/identity/repository/user_repo.go` - 用户仓储实现（已实现）
- `internal/domains/identity/errs/errors.go` - 领域错误定义（已实现）
- `internal/middleware/auth_context.go` - 认证拦截器，注入 user_id/tenant_id/role（已实现）
- `pkg/identityctx/identityctx.go` - Context 工具，读写认证信息（已实现）

### 9.2 前端文件

- 不适用（当前无前端实现）

---

## 10. 参考资料

- 用户故事：`docs/user-stories/01-platform-admin-user-stories.md`
- 用户故事：`docs/user-stories/02-tenant-admin-user-stories.md`
- 用户故事：`docs/user-stories/03-operator-user-stories.md`
- 角色定义：`docs/user-stories/_roles.md`
- 相关 PRD：`docs/prd/00-index.md`
- Proto 定义：`api/proto/identity/v1/user.proto`
- Proto 定义：`api/proto/identity/v1/auth.proto`
