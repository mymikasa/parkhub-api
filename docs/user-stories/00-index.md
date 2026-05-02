# 用户故事索引

> 角色统一定义：[`_roles.md`](_roles.md)

## 文件目录

| 文件 | 角色 / 主题 | 覆盖域 |
|------|------------|--------|
| [`01-platform-admin-user-stories.md`](01-platform-admin-user-stories.md) | 平台管理员 (Platform Admin) | identity |
| [`02-tenant-admin-user-stories.md`](02-tenant-admin-user-stories.md) | 租户管理员 (Tenant Admin) | identity |
| [`03-operator-user-stories.md`](03-operator-user-stories.md) | 操作员 (Operator) | identity |

## 故事 ID 规范

```
US-<DOMAIN>-<SEQ>
```

- `DOMAIN`: `AUTH`、`PARK`、`IOT`、`BILL`、`ORD`、`PAY`
- `SEQ`: 3 位递增数字，按域独立计数

示例：`US-AUTH-001`
