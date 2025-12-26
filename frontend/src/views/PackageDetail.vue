<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  ElCard,
  ElRow,
  ElCol,
  ElDescriptions,
  ElTag,
  ElButton,
  ElLink,
  ElIcon,
  ElSkeleton,
  ElAlert,
  ElMessage,
  ElTabs,
  ElTabPane,
  ElEmpty,
} from 'element-plus'
import { Document, Download, Link, CopyDocument, Box } from '@element-plus/icons-vue'
import { getPackageDetail, type PackageDetailResponse, type Package, type MetaData, type Dependency } from '../api/public'

const router = useRouter()
const route = useRoute()

const pkg = ref<PackageDetailResponse | null>(null)
const loading = ref(false)
const error = ref('')
const activeTab = ref('overview')

// 解析最新版本的 meta_data
const latestMetaData = computed<MetaData | null>(() => {
  if (!pkg.value?.versions?.length) return null
  const latest = pkg.value.versions[0]
  try {
    return JSON.parse(latest.meta_data)
  } catch {
    return null
  }
})

// 获取依赖列表
const dependencies = computed<Dependency[]>(() => {
  return latestMetaData.value?.index?.dependencies || []
})

// 是否有依赖
const hasDependencies = computed(() => {
  return dependencies.value.length > 0
})

const loadPackage = async () => {
  const name = route.params.name as string
  if (!name) {
    error.value = '包名不能为空'
    return
  }

  loading.value = true
  error.value = ''

  try {
    pkg.value = await getPackageDetail(name)
  } catch (err: any) {
    error.value = err.message || '加载包详情失败'
  } finally {
    loading.value = false
  }
}

const formatDate = (date: string) => {
  if (!date) return '未知'
  const d = new Date(date)
  if (isNaN(d.getTime())) return '未知'
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const copyToClipboard = async (text: string, successMsg: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(successMsg)
  } catch (err) {
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  loadPackage()
})
</script>

<template>
  <div class="package-detail-container">
    <div class="content-wrapper">
      <!-- Skeleton Loading -->
      <div v-if="loading" class="skeleton-wrapper">
        <el-skeleton animated>
          <template #template>
            <el-row :gutter="24">
              <el-col :span="16">
                <el-skeleton-item variant="rect" style="height: 200px; border-radius: 16px; margin-bottom: 24px;" />
                <el-skeleton-item variant="rect" style="height: 300px; border-radius: 16px;" />
              </el-col>
              <el-col :span="8">
                <el-skeleton-item variant="rect" style="height: 400px; border-radius: 16px;" />
              </el-col>
            </el-row>
          </template>
        </el-skeleton>
      </div>

      <!-- Error Alert -->
      <el-alert
        v-else-if="error"
        type="error"
        :title="error"
        show-icon
        :closable="false"
        class="error-alert"
      />

      <!-- Package Detail -->
      <div v-else-if="pkg" class="package-detail">
        <!-- Header -->
        <div class="header-card">
          <div class="package-header-content">
            <h1 class="package-name">{{ pkg.name }}</h1>
            <p class="package-description">{{ pkg.description || '暂无描述' }}</p>
          </div>
        </div>

        <!-- Main Content Layout -->
        <el-row :gutter="24">
          <!-- Left: Tabs Content -->
          <el-col :xs="24" :lg="16">
            <el-tabs v-model="activeTab" class="package-tabs">
              <!-- Overview Tab -->
              <el-tab-pane label="概览" name="overview">
                <!-- Installation Card -->
                <div class="info-card">
                  <h3 class="card-title">
                    <el-icon><Download /></el-icon>
                    安装方式
                  </h3>

                  <!-- TOML Config -->
                  <div class="install-block">
                    <div class="install-label">1. 在项目的 <code>cjpm.toml</code> 中添加依赖：</div>
                    <div class="config-box">
                      <pre><code>[dependencies]
{{ pkg.name }} = { version = '{{ pkg.versions[0]?.version }}' }</code></pre>
                      <el-button
                        size="small"
                        :icon="CopyDocument"
                        @click="copyToClipboard(`${pkg.name} = { version = '${pkg.versions[0]?.version}' }`, '配置已复制')"
                      >
                        复制
                      </el-button>
                    </div>
                  </div>

                  <!-- cjpm check -->
                  <div class="install-block">
                    <div class="install-label">2. 运行 cjpm 检查并下载依赖：</div>
                    <div class="command-box">
                      <code>cjpm check</code>
                      <el-button
                        size="small"
                        :icon="CopyDocument"
                        @click="copyToClipboard(`cjpm check`, '命令已复制')"
                      >
                        复制
                      </el-button>
                    </div>
                  </div>
                </div>

                <!-- Links Card -->
                <div v-if="pkg.homepage || pkg.repository" class="info-card">
                  <h3 class="card-title">
                    <el-icon><Link /></el-icon>
                    相关链接
                  </h3>
                  <div class="links-grid">
                    <el-link
                      v-if="pkg.homepage"
                      :href="pkg.homepage"
                      target="_blank"
                      class="link-item"
                    >
                      <el-icon><Link /></el-icon>
                      访问主页
                    </el-link>
                    <el-link
                      v-if="pkg.repository"
                      :href="pkg.repository"
                      target="_blank"
                      class="link-item"
                    >
                      <el-icon><Document /></el-icon>
                      查看仓库
                    </el-link>
                  </div>
                </div>
              </el-tab-pane>

              <!-- Dependencies Tab -->
              <el-tab-pane name="dependencies">
                <template #label>
                  <span>依赖</span>
                  <el-tag size="small" class="tab-count">{{ dependencies.length }}</el-tag>
                </template>

                <div class="info-card">
                  <el-empty v-if="!hasDependencies" description="暂无依赖" :image-size="80" />

                  <div v-else class="dependencies-list">
                    <div
                      v-for="dep in dependencies"
                      :key="dep.name"
                      class="dependency-item"
                    >
                      <div class="dependency-main">
                        <el-link
                          :href="`/packages/${dep.name}`"
                          class="dependency-name"
                          @click.prevent="router.push(`/packages/${dep.name}`)"
                        >
                          {{ dep.name }}
                        </el-link>
                        <el-tag size="small" type="info">{{ dep.require }}</el-tag>
                      </div>
                      <div v-if="dep.target || dep.type" class="dependency-meta">
                        <span v-if="dep.target" class="meta-tag">目标: {{ dep.target }}</span>
                        <span v-if="dep.type" class="meta-tag">类型: {{ dep.type }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </el-tab-pane>

              <!-- Versions Tab -->
              <el-tab-pane name="versions">
                <template #label>
                  <span>版本历史</span>
                  <el-tag size="small" class="tab-count">{{ pkg.versions.length }}</el-tag>
                </template>

                <div class="versions-card">
                  <div class="version-list">
                    <div
                      v-for="version in pkg.versions"
                      :key="version.id"
                      class="version-item"
                    >
                      <div class="version-main">
                        <div class="version-info">
                          <span class="version-number">{{ version.version }}</span>
                          <el-tag v-if="version.id === pkg.versions[0]?.id" size="small" type="success">最新</el-tag>
                        </div>
                        <div class="version-meta">
                          <span class="version-date">发布于 {{ formatDate(version.created_at) }}</span>
                        </div>
                      </div>
                      <div class="version-actions">
                        <el-button
                          size="small"
                          :icon="CopyDocument"
                          @click="copyToClipboard(`${pkg.name} = { version = '${version.version}' }`, '配置已复制')"
                        >
                          复制配置
                        </el-button>
                      </div>
                    </div>
                  </div>
                </div>
              </el-tab-pane>
            </el-tabs>
          </el-col>

          <!-- Right: Sidebar -->
          <el-col :xs="24" :lg="8">
            <!-- Info Card -->
            <div class="sidebar-card">
              <h3 class="sidebar-title">包信息</h3>

              <div class="info-list">
                <div class="info-item">
                  <span class="info-label">版本</span>
                  <el-tag type="success">{{ pkg.versions[0]?.version }}</el-tag>
                </div>
                <div class="info-item">
                  <span class="info-label">组织</span>
                  <el-tag type="primary">{{ pkg.versions[0]?.organization || '默认' }}</el-tag>
                </div>
                <div class="info-item">
                  <span class="info-label">类型</span>
                  <el-tag :type="pkg.versions[0]?.artifact_type === 'src' ? 'info' : 'warning'">
                    {{ pkg.versions[0]?.artifact_type === 'src' ? '源码' : '二进制' }}
                  </el-tag>
                </div>
                <div v-if="pkg.versions[0]?.executable" class="info-item">
                  <span class="info-label">可执行</span>
                  <el-tag type="danger">是</el-tag>
                </div>
                <div class="info-item">
                  <span class="info-label">创建时间</span>
                  <span class="info-value">{{ formatDate(pkg.versions[0]?.created_at) }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">更新时间</span>
                  <span class="info-value">{{ formatDate(pkg.versions[0]?.updated_at) }}</span>
                </div>
              </div>

              <el-button
                type="primary"
                size="large"
                class="install-button"
                @click="copyToClipboard(`${pkg.name} = { version = '${pkg.versions[0]?.version}' }`, '配置已复制到剪贴板')"
              >
                <el-icon style="margin-right: 5px;"><CopyDocument /></el-icon>
                复制依赖配置
              </el-button>
            </div>
          </el-col>
        </el-row>
      </div>
    </div>
  </div>
</template>

<style scoped>
.package-detail-container {
  width: 100%;
  min-height: 100vh;
  background: #f5f7fa;
}

.content-wrapper {
  max-width: 1400px;
  margin: 0 auto;
  padding: 40px 20px 60px;
}

/* Header Card */
.header-card {
  background: white;
  border-radius: 16px;
  padding: 28px;
  margin-bottom: 24px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.package-header-content {
  width: 100%;
}

.package-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.package-name {
  font-size: 32px;
  font-weight: 700;
  color: #303133;
  margin: 0;
}

.version-badge {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  border: none;
  color: white;
  font-weight: 600;
}

.package-description {
  font-size: 16px;
  color: #606266;
  line-height: 1.6;
  margin: 0 0 16px 0;
}

.package-meta {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.meta-label {
  color: #909399;
  font-weight: 500;
}

.meta-value {
  color: #303133;
  font-weight: 500;
}

/* Tabs */
.package-tabs {
  margin-top: 0;
}

.package-tabs :deep(.el-tabs__header) {
  margin: 0 0 24px 0;
}

.package-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.package-tabs :deep(.el-tabs__item) {
  font-size: 16px;
  font-weight: 500;
  color: #606266;
  padding: 0 20px;
}

.package-tabs :deep(.el-tabs__item.is-active) {
  color: #2563eb;
  font-weight: 600;
}

.package-tabs :deep(.el-tabs__active-bar) {
  background: linear-gradient(90deg, #2563eb 0%, #3b82f6 100%);
  height: 3px;
}

.tab-count {
  margin-left: 8px;
}

/* Versions Card */
.versions-card {
  background: white;
  border-radius: 16px;
  padding: 28px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

/* Version History */
.version-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.version-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  background: #f5f7fa;
  border-radius: 12px;
  gap: 24px;
  transition: all 0.3s ease;
}

.version-item:hover {
  background: #e8ecf1;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.version-main {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
}

.version-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.version-number {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.version-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.version-date {
  font-size: 14px;
  color: #909399;
}

.version-actions {
  display: flex;
  gap: 8px;
}

/* Info Card */
.info-card {
  background: white;
  border-radius: 16px;
  padding: 28px;
  margin-bottom: 24px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.card-title {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 20px 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.card-title .el-icon {
  color: #2563eb;
}

/* Dependencies List */
.dependencies-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dependency-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 12px;
  transition: all 0.3s ease;
}

.dependency-item:hover {
  background: #e8ecf1;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.dependency-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.dependency-name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  text-decoration: none;
  transition: color 0.3s ease;
}

.dependency-name:hover {
  color: #2563eb;
}

.dependency-meta {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.meta-tag {
  font-size: 12px;
  color: #909399;
  padding: 4px 8px;
  background: white;
  border-radius: 6px;
}

/* Links Grid */
.links-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.link-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 20px;
  background: #f5f7fa;
  border-radius: 12px;
  transition: all 0.3s ease;
  font-size: 15px;
  font-weight: 500;
}

.link-item:hover {
  background: linear-gradient(135deg, #2563eb 0%, #3b82f6 100%);
  color: white;
  transform: translateY(-2px);
}

/* Install Block */
.install-block {
  margin-bottom: 20px;
}

.install-block:last-child {
  margin-bottom: 0;
}

.install-label {
  font-size: 14px;
  font-weight: 600;
  color: #606266;
  margin-bottom: 10px;
}

.command-box {
  background: #1e1e1e;
  border-radius: 12px;
  padding: 16px 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.command-box code {
  font-family: 'Fira Code', 'Consolas', monospace;
  font-size: 14px;
  color: #a9b7c6;
  flex: 1;
}

.config-box {
  background: #1e1e1e;
  border-radius: 12px;
  padding: 16px 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.config-box pre {
  margin: 0;
  flex: 1;
}

.config-box code {
  font-family: 'Fira Code', 'Consolas', monospace;
  font-size: 14px;
  color: #a9b7c6;
  line-height: 1.8;
}

/* Sidebar Card */
.sidebar-card {
  background: white;
  border-radius: 16px;
  padding: 28px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  position: sticky;
  top: 20px;
}

.sidebar-title {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 20px 0;
  padding-bottom: 16px;
  border-bottom: 2px solid #f0f0f0;
}

.info-list {
  margin-bottom: 24px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f5f7fa;
}

.info-item:last-child {
  border-bottom: none;
}

.info-label {
  font-size: 14px;
  color: #909399;
}

.info-value {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.install-button {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  background: linear-gradient(135deg, #2563eb 0%, #3b82f6 100%);
  border: none;
}

.install-button:hover {
  background: linear-gradient(135deg, #5568d3 0%, #653a8b 100%);
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(102, 126, 234, 0.4);
}

/* Skeleton & Error */
.skeleton-wrapper {
  padding: 30px 20px;
}

.error-alert {
  margin: 30px 20px;
  border-radius: 12px;
}

/* Responsive */
@media (max-width: 992px) {
  .sidebar-card {
    position: static;
    margin-top: 24px;
  }
}

@media (max-width: 768px) {
  .package-name {
    font-size: 24px;
  }

  .package-description {
    font-size: 14px;
  }

  .package-tabs :deep(.el-tabs__item) {
    font-size: 14px;
    padding: 0 12px;
  }

  .links-grid {
    grid-template-columns: 1fr;
  }

  .version-item {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .version-main {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .command-box,
  .config-box {
    flex-direction: column;
    align-items: stretch;
  }

  .command-box code,
  .config-box code {
    word-break: break-all;
  }

  .info-card,
  .versions-card,
  .sidebar-card {
    padding: 20px;
  }

  .sidebar-card {
    margin-top: 24px;
  }
}
</style>
