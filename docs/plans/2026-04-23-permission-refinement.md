# 权限模型精炼计划

> **目标**: 清理权限模型，统一为"个人发布 + 团队管理"双路径
> **状态**: 已实现

## 设计背景

原设计试图完全用团队覆盖所有权限场景，但存在根本性问题：
1. 无组织包的首次发布者无法更新自己发布的包（因为没有建立团队关联）
2. TeamPackage 带有独立的 `permission` 字段，分散了权限控制点
3. 组织包与无组织包的规则描述复杂，不直观

## 核心设计：双路径权限模型

权限分两种独立路径，**任一满足即放行**：

```
                 ┌─────────────────────────┐
                 │    CheckPermission()     │
                 │   userID, org, pkgName,  │
                 │      requiredPerm        │
                 └──────────┬──────────────┘
                            │
              ┌─────────────┴─────────────┐
              ▼                           ▼
    ┌─────────────────┐       ┌───────────────────┐
    │  1. 个人发布路径  │       │   2. 团队管理路径   │
    │  (仅无组织包)     │       │  (有组织/无组织包)   │
    ├─────────────────┤       ├───────────────────┤
    │ publisher_id     │       │ user → team_members│
    │ 匹配即 write     │       │ team →关联(org/pkg)│
    │ (不支持overwrite)│       │ team.permission    │
    └─────────────────┘       └───────────────────┘
```

### 路径 1：个人发布（publisher_id 驱动）

适用范围：仅无组织包（organization=""）

| 操作 | 规则 |
|------|------|
| 发布新包 | 任意有效 Token 即可，记录 `publisher_id` |
| 发布新版本 | ✅ publisher 自动获得 `write` |
| 覆盖版本 | ❌ 拒绝（需走团队路径） |

原理：谁首次发布一个无组织包，谁就是该包的 publisher。publisher 可以自由发布新版本，但不能覆盖已有版本（覆盖需要团队 overwrite 权限）。

### 路径 2：团队管理（团队权限驱动）

适用范围：有组织包 + 无组织包

| 操作 | 规则 |
|------|------|
| 发布新包（有组织） | ❌ 拒绝，必需有团队关联了该组织 |
| 发布新版本 | ✅ 团队关联了该 org/pkg + team.permission >= `write` |
| 覆盖版本 | ✅ 团队关联了该 org/pkg + team.permission >= `overwrite` |

原理：团队通过两种方式获得包的管理权：
- **组织关联**（`team_organizations`）：团队关联一个组织后，对该组织下的所有包拥有 `team.permission` 级别的权限
- **包关联**（`team_packages`）：团队关联一个具体的 `(org, package_name)` 后，对该包拥有 `team.permission` 级别的权限

**关键简化：TeamPackage 不再有独立的 `permission` 字段**，一律走 `team.permission`。 `team_packages` 表简化为 `(team_id, organization, package_name)` 三元组。

## 权限矩阵

### 下载 / 索引查询

| 条件 | 结果 |
|------|------|
| `requireAuth=false` | ✅ 公开 |
| `requireAuth=true` + 有效 Token | ✅ 通过 |
| Token 无效/未提供 | ❌ 拒绝 |

### 无组织包——个人发布

| 条件 | 发布新包 | 发布新版本 | 覆盖版本 |
|------|---------|---------|---------|
| 任意有效 Token | ✅ 允许，记录 publisher | — | — |
| 自己是 publisher | — | ✅ write | ❌ 拒绝 |
| 团队关联该包 + team.permission >= write | — | ✅ | — |
| 团队关联该包 + team.permission >= overwrite | — | ✅ | ✅ |
| 超管 | ✅ | ✅ | ✅ |

### 有组织包——团队管理

| 条件 | 发布新包 | 发布新版本 | 覆盖版本 |
|------|---------|---------|---------|
| 任意有效 Token | ❌ 拒绝 | — | — |
| 团队关联该组织 + team.permission >= write | ✅ | ✅ | — |
| 团队关联该组织 + team.permission >= overwrite | ✅ | ✅ | ✅ |
| 团队关联具体包 + team.permission >= write | — | ✅ | — |
| 团队关联具体包 + team.permission >= overwrite | — | ✅ | ✅ |
| 超管 | ✅ | ✅ | ✅ |

## 变更清单

### Package 模型——加 publisher_id

`internal/models/package.go`:
```go
// 新增字段
PublisherID int64 `xorm:"'publisher_i_d' index" json:"publisher_id"`
```

首次发布时写入发起请求的用户 ID，记录包的"第一发布者"。

### TeamPackage 模型——删 permission

`internal/models/team.go`:
```go
// 删除字段
Permission string // 整行删除

// 改为：团队关联包只需 (team_id, organization, package_name)
// 权限级别统一走 team.permission
```

### PermissionChecker——加 publisher 检查 + 改 team_packages 检查

`internal/auth/permission.go`:

```go
func (p *PermissionChecker) CheckPermission(userID int64, org, pkgName, requiredPerm string) bool {
    if org == "" {
        // 路径 1：检查用户是否为 publisher
        var pkg models.Package
        has, _ := p.engine.Where("name = ? AND organization = ? AND publisher_i_d = ?",
            pkgName, org, userID).Get(&pkg)
        if has && PermissionLevel("write") >= PermissionLevel(requiredPerm) {
            return true
        }
    }

    // 路径 2：团队检查
    var members []models.TeamMember
    p.engine.Where("user_i_d = ?", userID).Find(&members)
    for _, member := range members {
        // 检查 team_packages（存在性匹配，权限走 team.permission）
        var count int64
        p.engine.Where("team_i_d = ? AND organization = ? AND package_name = ?",
            member.TeamID, org, pkgName).Count(&count)
        if count > 0 {
            var team models.Team
            p.engine.ID(member.TeamID).Get(&team)
            if PermissionLevel(team.Permission) >= PermissionLevel(requiredPerm) {
                return true
            }
        }

        // 检查 team_organizations（不变）
        ...
    }
    return false
}
```

### Publish Handler——记录 publisher

`internal/handlers/publish.go` 在插入 Package 记录时：
```go
pkg := &models.Package{
    // ...
    PublisherID: user.ID,  // 新增
}
```

发布新版本的权限检查不变（已走 CheckPermission）。

### Team Handler——去 permission

`internal/handlers/team.go`：

- `UpdateTeamPackages`：请求体去掉 `permission` 字段，只需 `organization` + `package_name`
- `ListTeamPackages`：返回体去掉 `permission`

### 前端——去 permission

`frontend/src/api/team.ts`：
- `TeamPackage` 接口去掉 `permission` 字段
- `updateTeamPackages` 参数去掉 `permission`

`frontend/src/views/admin/Teams.vue`：
- 包弹窗表格去掉"权限"列（el-select）
- 权限走团队的默认权限，在 UI 上标明"权限继承自团队：{team.permission}"

### 数据库迁移

`internal/migrations/v1.1.0_org_to_team.go`（合并后，原 v1.2.0 逻辑已并入）：
- `team_packages` 表删除 `permission` 列（幂等）
- `packages` 表加 `publisher_i_d` 列（默认 0 表示未知，幂等）
