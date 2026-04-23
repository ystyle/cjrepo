# 构建信息与关于弹窗 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在 Docker 构建时注入构建日期和 commit hash，在管理后台侧边栏底部添加"关于"菜单，点击显示弹窗展示构建信息和项目链接。

**Architecture:** 
1. 后端通过 Go ldflags 在编译时注入 buildDate/gitCommit/gitVersion 变量
2. 扩展 /api/stats API 返回构建信息
3. 前端创建 AboutDialog.vue 弹窗组件，在 AdminLayout.vue 底部添加菜单项

**Tech Stack:** Go ldflags, Fiber API, Vue 3, Element Plus Dialog

---

## Task 1: 后端添加构建信息变量

**Files:**
- Modify: `main.go:1-30`

**Step 1: 在 main.go 添加全局构建信息变量**

在 import 语句之前添加变量声明：

```go
package main

// 构建信息（通过 ldflags 注入）
var (
	buildDate   string = "unknown"
	gitCommit   string = "unknown"
	gitVersion  string = "dev"
)

import (
	"fmt"
	...
)
```

**Step 2: 提供获取构建信息的函数**

在变量声明后添加 getter 函数：

```go
// GetBuildInfo 返回构建信息
func GetBuildInfo() (buildDate, gitCommit, gitVersion string) {
	return buildDate, gitCommit, gitVersion
}
```

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: 添加构建信息全局变量和 getter 函数"
```

---

## Task 2: 扩展 stats API 返回构建信息

**Files:**
- Modify: `internal/handlers/public.go:26-63`

**Step 1: 扩展 StatsResponse 结构体**

修改 `StatsResponse` 添加构建信息字段：

```go
type StatsResponse struct {
	Packages     int64  `json:"packages"`
	Users        int64  `json:"users"`
	Versions     int64  `json:"versions"`
	Downloads    int64  `json:"downloads"`
	SiteName     string `json:"siteName"`
	BuildDate    string `json:"buildDate"`    // 新增
	GitCommit    string `json:"gitCommit"`    // 新增
	GitVersion   string `json:"gitVersion"`   // 新增
}
```

**Step 2: 修改 GetStats 函数返回构建信息**

修改 `GetStats` 函数，调用 main.GetBuildInfo()：

```go
func (h *PublicHandler) GetStats(c *fiber.Ctx) error {
	siteName := os.Getenv("CJREPO_SITE_NAME")
	if siteName == "" {
		siteName = "仓颉包仓库"
	}

	// ... 现有统计逻辑 ...

	// 获取构建信息
	buildDate, gitCommit, gitVersion := GetBuildInfo()

	return c.JSON(StatsResponse{
		Packages:     packageCount,
		Users:        userCount,
		Versions:     versionCount,
		Downloads:    downloadCount,
		SiteName:     siteName,
		BuildDate:    buildDate,
		GitCommit:    gitCommit,
		GitVersion:   gitVersion,
	})
}
```

**Step 3: 验证编译**

```bash
go build -o cjrepo main.go
```

**Step 4: Commit**

```bash
git add internal/handlers/public.go
git commit -m "feat: stats API 返回构建信息"
```

---

## Task 3: 修改 Dockerfile 添加 ldflags

**Files:**
- Modify: `Dockerfile:49-51`

**Step 1: 修改 Dockerfile 构建命令**

替换第 49-50 行的构建命令：

```dockerfile
# 构建应用（包含嵌入的前端文件和构建信息）
RUN BUILD_DATE=$(date -u '+%Y-%m-%d %H:%M:%S') && \
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    GIT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "dev") && \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -X main.buildDate=$BUILD_DATE -X main.gitCommit=$GIT_COMMIT -X main.gitVersion=$GIT_VERSION" \
    -o cjrepo .
```

**Step 2: Commit**

```bash
git add Dockerfile
git commit -m "feat: Docker 构建注入构建日期和 commit hash"
```

---

## Task 4: 前端扩展 API 类型定义

**Files:**
- Modify: `frontend/src/api/public.ts:4-10`

**Step 1: 扩展 Stats 接口**

修改 Stats 接口添加构建信息字段：

```typescript
export interface Stats {
  packages: number
  users: number
  versions: number
  downloads: number
  siteName: string
  buildDate: string    // 新增
  gitCommit: string    // 新增
  gitVersion: string   // 新增
}
```

**Step 2: Commit**

```bash
git add frontend/src/api/public.ts
git commit -m "feat: 前端 Stats 类型添加构建信息字段"
```

---

## Task 5: 创建 AboutDialog 组件

**Files:**
- Create: `frontend/src/components/AboutDialog.vue`

**Step 1: 创建 AboutDialog.vue 组件**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElDialog } from 'element-plus'
import { getStats } from '../api/public'
import type { Stats } from '../api/public'

const visible = ref(false)
const stats = ref<Stats | null>(null)

const open = () => {
  visible.value = true
  if (!stats.value) {
    loadStats()
  }
}

const loadStats = async () => {
  try {
    stats.value = await getStats()
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

defineExpose({ open })
</script>

<template>
  <ElDialog
    v-model="visible"
    title="关于 cjrepo"
    width="400px"
    :close-on-click-modal="true"
  >
    <div class="about-content">
      <h2 class="site-name">{{ stats?.siteName || '仓颉包仓库' }}</h2>
      
      <div class="version-info">
        <span class="version">{{ stats?.gitVersion || 'dev' }}</span>
        <span class="commit">({{ stats?.gitCommit || 'unknown' }})</span>
      </div>
      
      <div class="build-date">
        构建时间：{{ stats?.buildDate || 'unknown' }}
      </div>
      
      <hr class="divider" />
      
      <div class="links">
        <h4>项目链接</h4>
        <div class="link-item">
          <a href="https://github.com/anomalyco/cjrepo" target="_blank">
            GitHub 仓库
          </a>
        </div>
        <div class="link-item">
          <a href="https://cangjie-lang.cn" target="_blank">
            仓颉语言官网
          </a>
        </div>
        <div class="link-item">
          <a href="https://cangjie-lang.cn/docs" target="_blank">
            仓颉语言文档
          </a>
        </div>
      </div>
    </div>
  </ElDialog>
</template>

<style scoped>
.about-content {
  text-align: center;
}

.site-name {
  margin: 0 0 10px 0;
  font-size: 20px;
  color: #303133;
}

.version-info {
  margin-bottom: 8px;
  font-size: 14px;
}

.version {
  font-weight: bold;
  color: #409eff;
}

.commit {
  color: #909399;
  font-size: 12px;
}

.build-date {
  font-size: 13px;
  color: #606266;
  margin-bottom: 16px;
}

.divider {
  border: none;
  border-top: 1px solid #ebeef5;
  margin: 16px 0;
}

.links h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #606266;
}

.link-item {
  margin: 8px 0;
}

.link-item a {
  color: #409eff;
  text-decoration: none;
}

.link-item a:hover {
  text-decoration: underline;
}
</style>
```

**Step 2: Commit**

```bash
git add frontend/src/components/AboutDialog.vue
git commit -m "feat: 创建 AboutDialog 弹窗组件"
```

---

## Task 6: 修改 AdminLayout 添加"关于"菜单

**Files:**
- Modify: `frontend/src/layouts/AdminLayout.vue`

**Step 1: 导入 AboutDialog 组件**

在 script setup 部分添加导入：

```vue
<script setup lang="ts">
import { useRoute } from 'vue-router'
import {
  ElContainer,
  ElAside,
  ElMenu,
  ElMenuItem,
  ElHeader,
} from 'element-plus'
import { House, Box, Document, DataAnalysis, User, Setting, Connection, Log, Back, InfoFilled } from '@element-plus/icons-vue'
import { siteName } from '../stores/site'
import CjBox from '../components/CjBox.vue'
import AboutDialog from '../components/AboutDialog.vue'
import { ref } from 'vue'

const route = useRoute()
const aboutDialogRef = ref<InstanceType<typeof AboutDialog> | null>(null)

const openAbout = () => {
  aboutDialogRef.value?.open()
}
</script>
```

**Step 2: 在侧边栏底部添加"关于"菜单**

修改 el-menu 部分，将"返回首页"改为普通菜单项，在底部添加"关于"：

```vue
<el-menu
  :default-active="route.path"
  router
  class="el-menu"
>
  <el-menu-item index="/admin/dashboard">
    <el-icon><DataAnalysis /></el-icon>
    <span>仪表盘</span>
  </el-menu-item>
  <el-menu-item index="/admin/packages">
    <el-icon><Box /></el-icon>
    <span>包管理</span>
  </el-menu-item>
  <el-menu-item index="/admin/users">
    <el-icon><User /></el-icon>
    <span>用户管理</span>
  </el-menu-item>
  <el-menu-item index="/admin/organizations">
    <el-icon><Setting /></el-icon>
    <span>组织管理</span>
  </el-menu-item>
  <el-menu-item index="/admin/upstreams">
    <el-icon><Connection /></el-icon>
    <span>上游管理</span>
  </el-menu-item>
  <el-menu-item index="/admin/logs">
    <el-icon><Log /></el-icon>
    <span>操作日志</span>
  </el-menu-item>
  
  <!-- 底部菜单 -->
  <el-menu-item index="/" class="bottom-menu">
    <el-icon><House /></el-icon>
    <span>返回首页</span>
  </el-menu-item>
  <el-menu-item class="bottom-menu about-menu" @click="openAbout">
    <el-icon><InfoFilled /></el-icon>
    <span>关于</span>
  </el-menu-item>
</el-menu>

<!-- 关于弹窗 -->
<AboutDialog ref="aboutDialogRef" />
```

**Step 3: 添加底部菜单样式**

在 style 部分添加底部菜单样式：

```css
.bottom-menu {
  margin-top: auto;
}

.about-menu {
  cursor: pointer;
}
```

**Step 4: Commit**

```bash
git add frontend/src/layouts/AdminLayout.vue
git commit -m "feat: 侧边栏底部添加关于菜单和弹窗"
```

---

## Task 7: 构建并测试

**Step 1: 构建前端**

```bash
cd frontend && pnpm build && cd ..
```

**Step 2: 构建后端（本地测试）**

```bash
go build -o cjrepo main.go
```

**Step 3: 启动服务测试**

```bash
export CJREPO_ADMIN_KEY=test-key
./cjrepo
```

访问 http://localhost:8060/admin 登录后检查侧边栏底部是否有"关于"菜单。

**Step 4: 测试 API**

```bash
curl http://localhost:8060/api/stats
```

确认返回包含 buildDate, gitCommit, gitVersion 字段。

---

## Task 8: Docker 构建测试

**Step 1: 构建 Docker 镜像**

```bash
docker build -t cjrepo:test .
```

**Step 2: 运行容器测试**

```bash
docker run -e CJREPO_ADMIN_KEY=test-key -p 8060:8060 cjrepo:test
```

**Step 3: 验证构建信息**

```bash
curl http://localhost:8060/api/stats
```

确认 buildDate 和 gitCommit 包含实际值（非 "unknown"）。

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: 构建信息与关于弹窗功能完成"
```