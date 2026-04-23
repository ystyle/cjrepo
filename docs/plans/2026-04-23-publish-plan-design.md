# 发布计划功能设计文档

## 概述

为 cjrepo 添加发布计划功能，实现从私有仓批量发布到官方中心仓，解决多包项目的发布困难问题。

## 问题背景

**用户痛点**：
- 多包项目发布困难，需要逐个修改依赖版本
- 发布延迟不确定（几分钟到几十分钟）
- 需要等待索引更新才能继续发布下一个包
- 用户自己写脚本但维护成本高

**目标**：
- 自动分析依赖树，按正确顺序发布
- 自动等待索引更新
- 冲突检测与处理
- 简化多包发布流程

## 核心功能

### 1. 多起始包支持

**输入方式**：
- 输入框支持多个包（逗号分隔或换行）
- 格式：`组织::包名@版本` 或 `包名@版本`（无组织）

**示例**：
```
soulsoft::identity_tokens_jwt@1.1.0
soulsoft::extensions_http@1.1.0
asn1@1.0.0
```

### 2. 智能依赖分析

**流程**：
```
多个起始包 → 分别解析依赖树 → 合并去重 → 按依赖顺序排序
```

**去重优先级**：
- 同包名同版本 → 只保留一个
- 同包名不同版本 → 使用用户指定版本或最高版本

### 3. 版本对比与分类

**对比维度**：
- 版本号是否存在
- SHA256 是否一致

**分类结果**：

| 分类 | 条件 | 显示 | 处理建议 |
|------|------|------|---------|
| **conflict（冲突异常）** | 版本号相同 + SHA256 不同 | 🔴 冲突 | 警告，需 overwrite 权限重新覆盖或跳过 |
| **need_publish（需要发布）** | 中心仓无此版本 | 🔴 需要发布 | 正常发布 |
| **version可选（可选版本）** | 部分版本缺失（私有仓有多个版本符合依赖范围，中心仓部分缺失） | 🟡 可选发布 | 用户选择发布哪些版本 |
| **already_exists（已存在）** | 版本相同 + SHA256 相同 | ⚪ 已存在 | 跳过 |

### 4. 依赖版本范围解析

**场景示例**：
```
包 A 依赖 B: ">=1.0.0 <2.0.0"

私有仓 B 版本: [1.0.0, 1.1.0, 1.2.0, 1.3.0]
中心仓 B 版本: [1.0.0]

结果:
- 1.0.0: already_exists（SHA256 一致）
- 1.1.0: need_publish（中心仓缺失）
- 1.2.0: need_publish（中心仓缺失）
- 1.3.0: need_publish（中心仓缺失）

推荐: 1.3.0（私有仓最新符合版本）
```

### 5. 发布执行流程

**流程**：
```
用户确认发布计划 → 手动触发开始
  ↓
按依赖顺序逐个发布：
  1. POST 到中心仓 /pkg/{name}?organization={org}
  2. 轮询查询中心仓索引 /index/{first}/{second}/{name}
  3. 直到索引中出现该版本（SHA256 一致）
  4. 继续下一个包
  ↓
完成/失败时显示汇总
```

**轮询配置**：
- 间隔：可配置（默认 30 秒）
- 超时：可配置（默认 10 分钟）
- 失败处理：可暂停/跳过/继续

## 数据模型

```go
// PublishPlan 发布计划
type PublishPlan struct {
    ID          int64     `xorm:"pk autoincr 'i_d'" json:"id"`
    Name        string    `xorm:"not null" json:"name"`
    TargetUpstream int64   `xorm:"not null" json:"target_upstream"` // 目标上游（中心仓）
    Status      string    `xorm:"not null" json:"status"` // pending/running/completed/failed/paused
    CreatedAt   time.Time `xorm:"created" json:"created_at"`
    UpdatedAt   time.Time `xorm:"updated" json:"updated_at"`
}

// PublishPlanItem 计划项
type PublishPlanItem struct {
    ID          int64     `xorm:"pk autoincr 'i_d'" json:"id"`
    PlanID      int64     `xorm:"index not null" json:"plan_id"`
    Organization string   `xorm:"index" json:"organization"`
    PackageName  string   `xorm:"index not null" json:"package_name"`
    Version     string    `xorm:"not null" json:"version"`
    SHA256      string    `xorm:"not null" json:"sha256"` // 用于校验
    Order       int       `xorm:"not null" json:"order"` // 发布顺序
    Category    string    `xorm:"not null" json:"category"` // conflict/need_publish/version_optional/already_exists
    Status      string    `xorm:"not null" json:"status"` // pending/publishing/waiting_index/completed/failed/skipped
    Selected    bool      `xorm:"default false" json:"selected"` // 用户是否选择发布
    Error       string    `xorm:"text" json:"error"`
    StartedAt   time.Time `json:"started_at"`
    CompletedAt time.Time `json:"completed_at"`
}
```

## API 设计

### 分析依赖

| 方法 | 路径 | 说明 |
|------|-----|------|
| POST | `/api/admin/publish-plans/analyze` | 分析多个起始包的依赖，返回分类结果 |

**请求体**：
```json
{
  "packages": [
    {"organization": "soulsoft", "name": "identity_tokens_jwt", "version": "1.1.0"},
    {"organization": "", "name": "asn1", "version": "1.0.0"}
  ],
  "target_upstream": 1
}
```

**响应体**：
```json
{
  "packages": [
    {
      "organization": "soulsoft",
      "name": "identity_tokens",
      "version": "1.2.0",
      "sha256": "abc123...",
      "category": "need_publish",
      "dependency_range": ">=1.1.0",
      "local_versions": ["1.1.0", "1.2.0"],
      "remote_versions": [],
      "recommended_version": "1.2.0",
      "selected": true
    },
    {
      "organization": "",
      "name": "asn1",
      "version": "1.0.0",
      "sha256": "abc123...",
      "remote_sha256": "def456...",
      "category": "conflict",
      "selected": false
    }
  ],
  "publish_order": [
    {"organization": "soulsoft", "name": "identity_tokens", "version": "1.2.0"},
    {"organization": "soulsoft", "name": "identity_tokens_jwt", "version": "1.1.0"}
  ]
}
```

### 发布计划管理

| 方法 | 路径 | 说明 |
|------|-----|------|
| GET | `/api/admin/publish-plans` | 发布计划列表 |
| POST | `/api/admin/publish-plans` | 创建发布计划（body: `{name, target_upstream, items}`） |
| GET | `/api/admin/publish-plans/:id` | 获取计划详情 |
| PUT | `/api/admin/publish-plans/:id/items` | 更新计划项（用户选择） |
| POST | `/api/admin/publish-plans/:id/start` | 开始执行发布 |
| POST | `/api/admin/publish-plans/:id/pause` | 暂停发布 |
| POST | `/api/admin/publish-plans/:id/resume` | 继续发布 |
| DELETE | `/api/admin/publish-plans/:id` | 删除计划 |

### 上游配置

需要先配置官方中心仓作为 Upstream：
- URL: `https://pkg.cangjie-lang.cn`
- Publish Token: 用户需提供中心仓发布凭证

## 前端设计

### 新增页面

| 页面 | 路径 | 功能 |
|------|-----|------|
| PublishPlans.vue | `/admin/publish-plans` | 发布计划列表 |
| PublishPlanCreate.vue | `/admin/publish-plans/create` | 创建发布计划（多步骤） |
| PublishPlanDetail.vue | `/admin/publish-plans/:id` | 计划详情与执行状态 |

### 创建发布计划步骤

**步骤 1: 选择起始包**
- 多行输入框，支持批量输入
- 格式提示：`组织::包名@版本` 或 `包名@版本`

**步骤 2: 分析依赖**
- 显示分析结果，按分类分组
- 冲突异常组：显示 SHA256 差异，选择处理方式
- 需要发布组：默认勾选
- 可选版本组：用户勾选
- 已存在组：显示，不可勾选

**步骤 3: 确认发布顺序**
- 显示排序后的发布顺序
- 用户可调整顺序（拖拽或手动）

**步骤 4: 创建计划**
- 输入计划名称
- 确认创建

### 计划详情页

**显示内容**：
- 计划基本信息
- 执行进度条
- 包列表（状态实时更新）
- 操作按钮：开始/暂停/继续/删除

**状态显示**：
| 状态 | 显示 |
|------|------|
| pending | ⏳ 等待开始 |
| publishing | 🔄 发布中... |
| waiting_index | ⏳ 等待索引更新（轮询） |
| completed | ✅ 完成 |
| failed | ❌ 失败（显示错误） |
| skipped | ⏭️ 跳过 |

## 权限要求

- 需要 `write` 权限：发布新版本
- 需要 `overwrite` 权限：处理冲突（覆盖已存在版本）
- 超级管理员：可操作所有包

## 表名约定

| 表名 | 名称 |
|------|------|
| publish_plans | 发布计划 |
| publish_plan_items | 计划项 |

## 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| poll_interval | 30s | 索引轮询间隔 |
| poll_timeout | 10min | 单包超时时间 |
| max_concurrent | 1 | 并发发布数（暂时只支持串行） |