# 团队权限功能设计文档

## 概述

为 cjrepo 添加团队权限功能，实现精细化的包管理权限控制。

## 概念定义

| 概念 | 定义 |
|------|------|
| Organization（组织） | 客户/项目隔离，包归属组织。对标 npm scope、Maven groupId |
| Team（团队） | 权限管理单元，配置：成员 + 可操作组织/包 + 权限级别 |
| User（用户） | 通过加入团队获得权限，可加入多个团队 |

## 数据模型

### 新增表

```go
// Team 团队
type Team struct {
    ID            int64     `xorm:"pk autoincr"`
    Name          string    `xorm:"unique not null"`
    DisplayName   string    `xorm:"varchar(255)"`
    Description   string    `xorm:"text"`
    Permission    string    `xorm:"varchar(20) not null"` // read/write/overwrite
    CreatedAt     time.Time `xorm:"created"`
    UpdatedAt     time.Time `xorm:"updated"`
}

// TeamOrganization 团队可操作的组织
type TeamOrganization struct {
    ID             int64     `xorm:"pk autoincr"`
    TeamID         int64     `xorm:"index not null"`
    OrganizationID *int64    `xorm:"index"` // NULL 表示无组织包
    CreatedAt      time.Time `xorm:"created"`
}

// TeamPackage 团队对特定包的权限（覆盖组织级权限）
type TeamPackage struct {
    ID             int64     `xorm:"pk autoincr"`
    TeamID         int64     `xorm:"index not null"`
    Organization   string    `xorm:"index"`         // 组织名（空字符串表示无组织包）
    PackageName    string    `xorm:"index not null"` // 包名
    Permission     string    `xorm:"varchar(20) not null"` // read/write/overwrite
    CreatedAt      time.Time `xorm:"created"`
}

// TeamMember 团队成员
type TeamMember struct {
    ID        int64     `xorm:"pk autoincr"`
    TeamID    int64     `xorm:"index not null"`
    UserID    int64     `xorm:"index not null"`
    CreatedAt time.Time `xorm:"created"`
}
```

### 删除表

- **OrganizationMember**：组织成员通过团队间接管理

### 权限级别

| 权限 | 能力 | 递进关系 |
|------|------|---------|
| read | 索引查询、下载包 | 基础级别 |
| write | read + 发布新版本 | 包含 read |
| overwrite | write + 覆盖已发布版本 | 最高级别 |

## 权限检查

### 操作与权限要求

| 操作 | API | 所需权限 |
|------|-----|---------|
| 索引查询 | `/index/{first}/{second}/{name}` | read |
| 下载包 | `/pkg/{name}/{version}` | read |
| 发布新版本 | `/pkg/{name}?organization={org}` | write |
| 覆盖已存在版本 | `/pkg/{name}?organization={org}` | overwrite |

### 权限检查流程

```
1. 用户请求 → 提取 Token → 获取用户 ID
2. 超级管理员（IsSuperuser）直接通过
3. 查用户所属团队 → 遍历团队检查：
   a. 先查 TeamPackage（包级权限）
   b. 无包级权限 → 查 TeamOrganization（组织级权限）
4. 权限级别满足要求 → 通过；否则 403
```

### 发布时特殊处理

- 版本已存在 → 需要 `overwrite` 权限
- 版本不存在 → 需要 `write` 权限

## API 设计

### 团队管理

| 方法 | 路径 | 说明 |
|------|-----|------|
| GET | `/api/admin/teams` | 团队列表 |
| POST | `/api/admin/teams` | 创建团队 |
| PUT | `/api/admin/teams/:id` | 更新团队基本信息 |
| DELETE | `/api/admin/teams/:id` | 删除团队 |

### 团队关联（Replace 模式）

| 方法 | 路径 | 说明 |
|------|-----|------|
| PUT | `/api/admin/teams/:id/organizations` | 替换关联组织（body: `{organization_ids: [1, null, 2]}`） |
| PUT | `/api/admin/teams/:id/packages` | 替换包权限（body: `{packages: [{organization, package_name, permission}]}`） |
| PUT | `/api/admin/teams/:id/members` | 替换成员（body: `{user_ids: [1, 2, 3]}`） |

### 用户团队查询

| 方法 | 路径 | 说明 |
|------|-----|------|
| GET | `/api/admin/users/:id/teams` | 用户所属团队列表 |

## 前端设计

### 新增页面

| 页面 | 路径 | 功能 |
|------|-----|------|
| Teams.vue | `/admin/teams` | 团队列表，操作列分开 |

### 新增弹窗组件

| 组件 | 功能 |
|------|------|
| TeamFormDialog.vue | 团队基本信息（名称、描述） |
| TeamOrganizationsDialog.vue | 分配组织，多选 + 无组织选项 |
| TeamPackagesDialog.vue | 分配包权限，表格添加/编辑/删除 |
| TeamMembersDialog.vue | 管理成员，多选用户 |

### 操作列按钮（分开操作）

- 编辑：弹出 TeamFormDialog
- 分配组织：弹出 TeamOrganizationsDialog
- 分配包：弹出 TeamPackagesDialog
- 管理成员：弹出 TeamMembersDialog
- 删除：确认后删除

### 调整现有页面

| 页面 | 调整 |
|------|------|
| Users.vue | 显示用户所属团队 |
| Organizations.vue | 显示关联团队 |

### 侧边栏菜单

新增「团队管理」菜单项，与组织管理并列。

## 表名约定

| 表 | 表名 |
|---|------|
| Team | teams |
| TeamOrganization | team_organizations |
| TeamPackage | team_packages |
| TeamMember | team_members |