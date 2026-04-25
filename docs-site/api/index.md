# API

CJRepo 兼容 cjpm 的发布/下载/索引协议，协议细节直接参考官方文档：

| 协议 | 官方文档 |
|------|---------|
| 发布/下载/索引协议 | [中心仓通信规格](https://pkgdocs.cangjie-lang.cn/docs/zh/1.0.0/central-repo/source_zh_cn/appendix/api.html) |
| 元数据格式 | [meta-data.json 规范](https://pkgdocs.cangjie-lang.cn/docs/zh/1.0.0/central-repo/source_zh_cn/appendix/meta_data.html) |

> CJRepo 的 `docs/中心仓通信规格.md` 中也有一份本地存档，但以官方文档为准。

## 管理 API

CJRepo 私有管理 API 用于后台管理，需要 JWT 认证：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/admin/login` | 管理员登录 |
| GET | `/api/admin/dashboard` | 仪表盘统计 |
| GET | `/api/admin/packages` | 包列表（分页） |
| GET | `/api/admin/users` | 用户列表（分页+搜索） |
| POST | `/api/admin/users` | 创建用户 |
| GET | `/api/admin/organizations` | 组织列表（分页+搜索） |
| POST | `/api/admin/organizations` | 创建组织 |
| GET | `/api/admin/teams` | 团队列表（分页） |
| POST | `/api/admin/teams` | 创建团队 |
| GET | `/api/admin/upstreams` | 上游源列表 |
| GET | `/api/admin/logs/publish` | 发布日志 |
| GET | `/api/admin/logs/admin` | 操作日志 |

## 认证

**cjpm 协议端点**：通过 `Authorization` 头传递用户 Token：

```
Authorization: <user_token>
```

**管理 API**：先登录获取 JWT Token，再通过 `Bearer` 方式访问：

```
Authorization: Bearer <jwt_token>
```
