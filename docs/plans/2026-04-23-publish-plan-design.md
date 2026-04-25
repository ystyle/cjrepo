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
- 输入框支持多个包（select mode=tag/muti）
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
    ID        int64     `xorm:"pk autoincr 'i_d'" json:"id"`
    PlanID    int64     `xorm:"index not null" json:"plan_id"`
    PackageID int64     `xorm:"index not null" json:"package_id"` // 关联 packages 表（确定 org+name+version+sha256）
    Order     int       `xorm:"not null" json:"order"` // 发布顺序
    Category  string    `xorm:"not null" json:"category"` // conflict/need_publish/version_optional/already_exists
    Status    string    `xorm:"not null" json:"status"` // pending/publishing/waiting_index/completed/failed/skipped
    Selected  bool      `xorm:"default false" json:"selected"` // 用户是否选择发布
    Error     string    `xorm:"text" json:"error"`
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
      "package_id": 42,
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
      "package_id": 18,
      "organization": "",
      "name": "asn1",
      "version": "1.0.0",
      "sha256": "abc123...",
      "remote_sha256": "def456...",
      "category": "conflict",
      "selected": false
    }
  ],
  "publish_order": [42, 43]
}
```

### 发布计划管理

| 方法 | 路径 | 说明 |
|------|-----|------|
| GET | `/api/admin/publish-plans` | 发布计划列表 |
| POST | `/api/admin/publish-plans` | 创建发布计划（body: `{name, target_upstream, package_ids}`） |
| GET | `/api/admin/publish-plans/:id` | 获取计划详情 |
| PUT | `/api/admin/publish-plans/:id/items` | 更新计划项选择状态（body: `{selected_ids: [42, 43]}`） |
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

## 后台任务执行

### 方案

自建 `TaskManager` 全局单例，不依赖外部 worker 或消息队列。

```
TaskManager
  ├─ Start(planID)      → 启动 goroutine，按序发布
  ├─ Pause(planID)      → 通过 channel 暂停
  ├─ Resume(planID)     → 继续执行
  ├─ Status(planID)     → 查询当前进度（从 DB）
  └─ 内部 goroutine：
      读取计划项 → 逐项发布 → 轮询索引 → 下一项
```

### 执行顺序

```
for each item (按 order 排序):
  1. POST /pkg/{name}?organization={org}  发布到目标上游
  2. 轮询上游索引，直到出现该版本的记录（SHA256 一致）
  3. 更新 DB 状态为 completed
  4. 下一项
```

### 暂停/恢复

- `Pause()`：通过 context cancel + channel 通知 goroutine 停止
- `Resume()`：重新启动 goroutine，从当前 `pending` 状态的第一项继续
- 服务重启：启动时扫描 DB，将 `running` 状态的计划标记为 `paused`

### 状态流转

```
pending → running → completed
                  → failed（可手动重试）
                  → paused（手动暂停/服务重启）
        paused  → running（继续）
```

### 为什么不自建 worker

- 场景单一：串行执行，一对多
- 并发量小：用户一次只操作一个计划
- 需要暂停/恢复：channel 控制比消息队列简单
- 持久化：DB 状态已够，不需要额外存储

## UI 设计

三页设计，全页面布局（非弹窗），与现有管理后台风格一致。

### 列表页 `/admin/publish-plans`

```
┌──────────────────────────────────────────────────────────────────┐
│  发布计划                                               ┌─────┐ │
│                                                        │ +新建│ │
│                                                        └─────┘ │
│ ┌──────┬─────────┬──────┬─────────┬──────────┬──────────┐     │
│ │ 名称 │ 目标上游 │ 包数 │ 状态    │ 创建时间  │ 操作     │     │
│ ├──────┼─────────┼──────┼─────────┼──────────┼──────────┤     │
│ │v1.1  │ 官方中心│ 12   │ ✅ 完成 │04-25 14 │ 查看 删除 │     │
│ │      │ 仓      │      │         │:00      │          │     │
│ ├──────┼─────────┼──────┼─────────┼──────────┼──────────┤     │
│ │hotfix│ 官方中心│ 3    │ ⏳ 暂停 │04-24 10 │ 查看 删除 │     │
│ │      │ 仓      │      │         │:30      │          │     │
│ └──────┴─────────┴──────┴─────────┴──────────┴──────────┘     │
│                                                 ┌──┐           │
│  1  2  3 ... 10                                 │20│/页        │
│                                                 └──┘           │
└──────────────────────────────────────────────────────────────────┘
```

#### 列表页说明
- 标准表格，与现有用户管理/团队管理一致的分页
- 状态列带颜色标签：完成=绿色、运行中=蓝色、暂停=橙色、失败=红色
- 操作列：查看（进入详情）、删除（确认后删除）

### 创建页 `/admin/publish-plans/create`

```
┌──────────────────────────────────────────────────────────────────┐
│  新建发布计划                                      ← 返回列表    │
│                                                     当前: 步骤 1  │
│  ●━━━━━━━○━━━━━━━○                                               │
│  选择包    分析     确认                                           │
│  步骤 1 ── 步骤 2 ── 步骤 3                                      │
│                                                                   │
│  ┌─ 起始包 ────────────────────────────────────────────────────┐ │
│  │                                                               │ │
│  │  目标上游: [官方中心仓 ▾]  ← 必选，决定发布到哪里和对比版本  │ │
│  │                                                               │ │
│  │ ┌────────────────────────────────────────────────────────┐ │ │
│  │ │ 输入包名，格式: org::name@version 或 name@version  ↵  │ │ │
│  │ └────────────────────────────────────────────────────────┘ │ │
│  │ ┌──────────────────────────────────────────────────────┐ │ │
│  │ │ ◆ soulsoft::identity_tokens_jwt@1.1.0            ✕  │ │ │
│  │ │ ◆ soulsoft::extensions_http@1.1.0                ✕  │ │ │
│  │ │ ◆ asn1@1.0.0                                     ✕  │ │ │
│  │ └──────────────────────────────────────────────────────┘ │ │
│  │                               ┌──────────────┐            │ │
│  │                               │  分析依赖 →  │            │ │
│  │                               └──────────────┘            │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌─ 分析结果 ──────────────────────────────────────────────────┐ │
│  │ 🔴 冲突 (1)              🟡 可选版本 (3)                     │ │
│  │ ┌────────────────┐       ┌────────────────┐                  │ │
│  │ │ ☐ asn1@1.0.0   │       │ ☑ utils@1.2.0  │                  │ │
│  │ │   SHA256 不一致  │       │ ☑ utils@1.1.0  │                  │ │
│  │ └────────────────┘       │ ☐ core@2.0.1   │                  │ │
│  │                           └────────────────┘                  │ │
│  │ 🔴 需要发布 (5)          ⚪ 已存在 (2)                        │ │
│  │ ┌────────────────┐       ┌────────────────┐                  │ │
│  │ │ ☑ tokens@1.1.0 │       │ ○ base@1.0.0   │                  │ │
│  │ │ ☑ http@1.1.0   │       │ ○ log@2.1.0    │                  │ │
│  │ └────────────────┘       └────────────────┘                  │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌─ 发布顺序 ──────────────────────────────────────────────────┐ │
│  │  ┌──╥──────────────────────────────┐                        │ │
│  │  │1 ║ tokens@1.1.0        ⠿ ☐  ║  │                        │ │
│  │  │2 ║ http@1.1.0          ⠿ ☐  ║  │                        │ │
│  │  │3 ║ utils@1.2.0         ⠿ ☐  ║  │                        │ │
│  │  └──╨──────────────────────────────┘                        │ │
│  │  ⠿ = 可拖拽排序                                                   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                          ┌──────┐ ┌──────┐     │
│                                          │ 上一步│ │创建计划│     │
│                                          └──────┘ └──────┘     │
└──────────────────────────────────────────────────────────────────┘
```

#### 分析阶段 UI

点击「分析依赖」后，不转圈干等，显示实时进度：

```
┌─ 正在分析依赖 ─────────────────────────────────────┐
│                                                     │
│  ████████████░░░░░░░░░░░░░░░░░░  40%               │
│                                                     │
│  ✓ tokens@1.1.0          → 3 个依赖                 │
│  ✓ http@1.1.0            → 5 个依赖                 │
│  ⟳ core@2.0.1            → 正在获取远程版本对比...  │
│  ○ utils@1.2.0           → 等待中                   │
│                                                     │
├─────────────────────────────────────────────────────┤
│  ✅ 分析完成，发现 12 个包需要处理                   │
└─────────────────────────────────────────────────────┘
```

实现方式：前端提交分析请求后，每隔 1-2 秒轮询 GET `/api/admin/publish-plans/analyze/:taskId` 获取当前进度，不做 SSE（分析是一次性操作，非持续流）。

#### 创建页说明
- 三步向导：选起始包 → 分析结果 → 确认顺序
- 步骤 1：顶部选择目标上游（即发布到哪个中心仓），然后输入起始包
- 步骤 2：分析结果按分类分组显示，用户勾选要发布的包
- 步骤 3：确认发布顺序，支持拖拽调整

### 详情页 `/admin/publish-plans/:id`

```
┌──────────────────────────────────────────────────────────────────┐
│  v1.1 发布计划                                       ← 返回列表 │
│                                                                   │
│  ┌─ 基本信息 ──────────────────────────────────────────────────┐ │
│  │  名称: v1.1   上游: 官方中心仓   状态: ✅ 完成               │ │
│  │  创建: 2026-04-25 14:00         完成: 2026-04-25 14:12      │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  进度: 12/12  ████████████████████████████████████  100%         │
│                                                                   │
│  ┌─ 发布项 ─────────────────────────────────────────────────────┐│
│  │ # │ 包                    │ 版本  │ 耗时   │ 状态           ││
│  │ 1 │ soulsoft::tokens      │ 1.0.0 │ 0:30  │ ✅ 已发布       ││
│  │ 2 │ soulsoft::http        │ 1.1.0 │ 0:45  │ ✅ 已发布       ││
│  │ 3 │ utils                 │ 1.2.0 │ 1:10  │ ✅ 已发布       ││
│  │ 4 │ soulsoft::jwt         │ 1.1.0 │ 0:50  │ ✅ 已发布       ││
│  │ 5 │ core                  │ 2.0.1 │       │ ⏳ 等待索引...   ││
│  │ 6 │ ...                                                        ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌─ 执行日志 ───────────────────────────────────────────────────┐│
│  │ ✅ tokens@1.0.0 → 发布成功 (12:00)                           ││
│  │ ⏳ tokens@1.0.0 → 等待索引更新... (12:01)                    ││
│  │ ✅ tokens@1.0.0 → 索引已确认 (12:02)                         ││
│  │ ✅ http@1.1.0 → 发布成功 (12:03)                             ││
│  │ ❌ utils@1.2.0 → 发布失败: 超时 (12:05)                      ││
│  └──────────────────────────────────────────────────────────────┘│
│                                                                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                         │
│  │   暂停   │ │   继续   │ │   删除   │                         │
│  └──────────┘ └──────────┘ └──────────┘                         │
└──────────────────────────────────────────────────────────────────┘
```

#### 执行日志实时推送

详情页的执行日志使用 **SSE**（Server-Sent Events）实现实时推送。

```
GET /api/admin/publish-plans/:id/events
Accept: text/event-stream

data: {"type":"progress","current":5,"total":12}
data: {"type":"log","message":"✅ tokens@1.0.0 → 发布成功","time":"12:00"}
data: {"type":"status","status":"running"}
```

前端用 `EventSource` 监听，日志窗口自动滚动到底部。SSE 比短轮询更适合执行阶段：持续推送，延迟低，浏览器原生支持断连重试。

#### 详情页说明
- 顶部基本信息卡片
- 进度条：已发布数/总数，百分比
- 发布项列表：显示每包的状态，实时更新
- 执行日志：滚动日志，每条有时间戳
- 操作按钮随状态变化：
  - 运行中：暂停
  - 已暂停：继续
  - 已完成/失败：删除

## 实现状态

| 任务 | 模块 | 状态 |
|------|------|------|
| Models: PublishPlan + PublishPlanItem | ✅ | |
| DB Sync + Migration | ✅ | |
| TaskManager 框架（Start/Pause/Resume/Init） | ✅ | |
| Handlers: CRUD + Analyze + Events(SSE) | ✅ | |
| Routes 注册 | ✅ | |
| 前端列表页 PublishPlans.vue | ✅ | |
| 前端创建页 PublishPlanCreate.vue（三步向导） | ✅ | |
| 前端详情页 PublishPlanDetail.vue（SSE 日志） | ✅ | |
| 路由 + 侧边栏菜单 | ✅ | |
| Upstream 模块添加 PublishPackage 方法 | ✅ | |
| 修复上游索引 URL 格式（name[1] → name[2:4]）| ✅ | |
| TaskManager.publishToUpstream 对接 Upstream | ✅ | |
| TaskManager.pollIndex 上游索引轮询 | ✅ | |
| 版本范围解析库（支持区间/多组合） | ✅ | internal/semverutil |
| 分析阶段完整依赖树解析（本地） | ✅ | 从 meta_data 递归解析依赖、拓扑排序、版本匹配 |
| 版本对比（远程 vs 本地 SHA256） | ✅ | 分析阶段 fetch upstream index 比对 SHA256 |
| 计划级别可配置轮询间隔/超时 | ✅ | PublishPlan 加 poll_interval/poll_timeout 字段 |

## 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| poll_interval | 30s | 索引轮询间隔 |
| poll_timeout | 10min | 单包超时时间 |
| max_concurrent | 1 | 并发发布数（暂时只支持串行） |