<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  ElCard,
  ElRow,
  ElCol,
  ElInput,
  ElSelect,
  ElButton,
  ElPagination,
  ElTag,
  ElEmpty,
  ElIcon,
  ElSkeleton,
} from 'element-plus'
import { Search, Box } from '@element-plus/icons-vue'
import { getPackages, getOrganizations, type Package } from '../api/public'

const router = useRouter()

const packages = ref<Package[]>([])
const organizations = ref<string[]>([])
const loading = ref(false)

const searchQuery = ref('')
const selectedOrg = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

const loadOrganizations = async () => {
  try {
    const data = await getOrganizations()
    organizations.value = data.map((org) => org.name)
  } catch (error) {
    console.error('Failed to load organizations:', error)
  }
}

const loadPackages = async () => {
  loading.value = true
  try {
    const data = await getPackages({
      page: currentPage.value,
      pageSize: pageSize.value,
      search: searchQuery.value,
      org: selectedOrg.value,
    })
    packages.value = data.data
    total.value = data.total
  } catch (error) {
    console.error('Failed to load packages:', error)
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  loadPackages()
}

const handlePageChange = (page: number) => {
  currentPage.value = page
  loadPackages()
}

const goToDetail = (pkg: Package) => {
  router.push(`/packages/${pkg.name}`)
}

watch(selectedOrg, () => {
  currentPage.value = 1
  loadPackages()
})

onMounted(() => {
  loadOrganizations()
  loadPackages()
})
</script>

<template>
  <div class="packages-container">
    <!-- Search Section -->
    <section class="search-section">
      <div class="content-wrapper">
        <div class="section-header">
          <h1 class="section-title">发现仓颉包</h1>
          <p class="section-subtitle">浏览和搜索仓颉生态系统中的精彩包</p>
        </div>

        <div class="search-box">
          <el-input
            v-model="searchQuery"
            placeholder="搜索包名或描述..."
            size="large"
            clearable
            class="search-input"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
            <template #append>
              <el-button :icon="Search" @click="handleSearch">搜索</el-button>
            </template>
          </el-input>

          <div class="filters">
            <el-select
              v-model="selectedOrg"
              placeholder="筛选组织"
              clearable
              filterable
              size="large"
              class="org-select"
              @change="handleSearch"
            >
              <el-option
                v-for="org in organizations"
                :key="org"
                :label="org || '默认组织'"
                :value="org"
              />
            </el-select>
          </div>
        </div>
      </div>
    </section>

    <!-- Packages Grid -->
    <section class="packages-section">
      <div class="content-wrapper">
        <div v-if="total > 0" class="results-info">
          <span class="count">{{ total }}</span> 个包
          <span v-if="searchQuery || selectedOrg" class="filters-info">
            <span v-if="searchQuery">包含 "{{ searchQuery }}"</span>
            <span v-if="selectedOrg">在 {{ selectedOrg }} 组织</span>
          </span>
        </div>

        <el-skeleton v-if="loading" :loading="loading" animated>
          <template #template>
            <el-row :gutter="24">
              <el-col v-for="i in 8" :key="i" :xs="24" :sm="12" :md="8" :lg="6">
                <el-skeleton-item variant="rect" style="height: 280px; border-radius: 16px; margin-bottom: 24px;" />
              </el-col>
            </el-row>
          </template>
        </el-skeleton>

        <el-empty v-else-if="packages.length === 0" description="没有找到匹配的包" />

        <el-row v-else :gutter="24">
          <el-col
            v-for="pkg in packages"
            :key="pkg.id"
            :xs="24"
            :sm="12"
            :md="8"
            :lg="6"
            class="package-col"
          >
            <div class="package-card" @click="goToDetail(pkg)">
              <div class="package-content">
                <div class="package-header">
                  <h3 class="package-name">{{ pkg.name }}</h3>
                  <div class="package-tags">
                    <el-tag size="small" type="success" class="version-tag">
                      {{ pkg.version }}
                    </el-tag>
                    <el-tag size="small" class="org-tag">
                      {{ pkg.organization || '默认' }}
                    </el-tag>
                  </div>
                </div>
                <p class="package-description">{{ pkg.description || '暂无描述' }}</p>
                <div class="package-meta">
                  <el-tag size="small" :type="pkg.artifact_type === 'src' ? 'info' : 'warning'">
                    {{ pkg.artifact_type === 'src' ? '源码' : '二进制' }}
                  </el-tag>
                  <el-tag v-if="pkg.executable" size="small" type="danger">
                    可执行
                  </el-tag>
                </div>
              </div>
            </div>
          </el-col>
        </el-row>

        <!-- Pagination -->
        <div v-if="total > 0" class="pagination-wrapper">
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[12, 24, 48, 96]"
            :total="total"
            layout="total, sizes, prev, pager, next, jumper"
            background
            @current-change="handlePageChange"
            @size-change="loadPackages"
          />
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.packages-container {
  width: 100%;
  min-height: 100vh;
  background: #f5f7fa;
}

/* Search Section */
.search-section {
  background: white;
  padding: 40px 20px 30px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.section-header {
  text-align: center;
  margin-bottom: 30px;
}

.section-title {
  font-size: 32px;
  font-weight: 700;
  color: #303133;
  margin: 0 0 12px 0;
}

.section-subtitle {
  font-size: 16px;
  color: #909399;
  margin: 0;
}

.content-wrapper {
  max-width: 1400px;
  margin: 0 auto;
}

.search-box {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
}

.search-input {
  flex: 1;
  min-width: 300px;
}

.search-input :deep(.el-input__wrapper) {
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  padding: 8px 16px;
}

.search-input :deep(.el-input__wrapper:hover),
.search-input :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 4px 20px rgba(102, 126, 234, 0.2);
}

.filters {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.org-select {
  min-width: 200px;
}

.org-select :deep(.el-select__wrapper) {
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

/* Packages Section */
.packages-section {
  padding: 40px 20px 60px;
}

.results-info {
  margin-bottom: 24px;
  font-size: 15px;
  color: #606266;
}

.count {
  font-weight: 600;
  color: #2563eb;
  font-size: 18px;
}

.filters-info {
  color: #909399;
  margin-left: 8px;
}

.package-col {
  margin-bottom: 24px;
}

.package-card {
  background: white;
  border-radius: 16px;
  padding: 24px;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  height: 100%;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

.package-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, #2563eb 0%, #3b82f6 100%);
  transform: scaleX(0);
  transition: transform 0.3s ease;
}

.package-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.package-card:hover::before {
  transform: scaleX(1);
}

.package-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.package-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.package-name {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  margin: 0;
  line-height: 1.4;
}

.package-tags {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.package-description {
  font-size: 14px;
  color: #606266;
  line-height: 1.6;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.version-tag {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  border: none;
  color: white;
  font-weight: 500;
}

.org-tag {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
  border: none;
  color: white;
  font-weight: 500;
}

.package-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 4px;
}

/* Pagination */
.pagination-wrapper {
  margin-top: 40px;
  display: flex;
  justify-content: center;
  padding: 20px 0;
}

.pagination-wrapper :deep(.el-pagination) {
  gap: 8px;
}

.pagination-wrapper :deep(.el-pagination.is-background .el-pager li) {
  border-radius: 8px;
  font-weight: 500;
}

.pagination-wrapper :deep(.el-pagination.is-background .btn-next),
.pagination-wrapper :deep(.el-pagination.is-background .btn-prev) {
  border-radius: 8px;
}

/* Responsive */
@media (max-width: 768px) {
  .page-title {
    font-size: 28px;
  }

  .title-icon {
    font-size: 28px;
  }

  .page-subtitle {
    font-size: 16px;
  }

  .search-box {
    flex-direction: column;
    align-items: stretch;
  }

  .search-input {
    min-width: 100%;
  }

  .org-select {
    min-width: 100%;
  }
}
</style>
