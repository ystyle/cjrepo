<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  ElCard,
  ElTable,
  ElTableColumn,
  ElTag,
  ElButton,
  ElIcon,
  ElInput,
  ElSelect,
  ElRow,
  ElCol,
  ElTabs,
  ElTabPane,
  ElMessage,
} from 'element-plus'
import { Refresh, Document } from '@element-plus/icons-vue'
import { getPublishLogs, getAdminLogs, type PublishLog, type AdminLog } from '../../api/admin'

const activeTab = ref('publish')

// 发布日志
const publishLogs = ref<PublishLog[]>([])
const publishLoading = ref(false)
const publishTotal = ref(0)
const publishPage = ref(1)

// 管理员操作日志
const adminLogs = ref<AdminLog[]>([])
const adminLoading = ref(false)
const adminTotal = ref(0)
const adminPage = ref(1)

// 搜索筛选
const selectedStatus = ref('')
const selectedAction = ref('')

const loadPublishLogs = async () => {
  publishLoading.value = true
  try {
    const data = await getPublishLogs({
      page: publishPage.value,
      pageSize: 20,
      status: selectedStatus.value,
    })
    publishLogs.value = data.data || []
    publishTotal.value = data.total
  } catch (error: any) {
    ElMessage.error(error.message || '加载发布日志失败')
  } finally {
    publishLoading.value = false
  }
}

const loadAdminLogs = async () => {
  adminLoading.value = true
  try {
    const data = await getAdminLogs({
      page: adminPage.value,
      pageSize: 20,
      action: selectedAction.value,
    })
    adminLogs.value = data.data || []
    adminTotal.value = data.total
  } catch (error: any) {
    ElMessage.error(error.message || '加载操作日志失败')
  } finally {
    adminLoading.value = false
  }
}

const handleTabChange = () => {
  if (activeTab.value === 'publish') {
    loadPublishLogs()
  } else {
    loadAdminLogs()
  }
}

const handleSearch = () => {
  if (activeTab.value === 'publish') {
    publishPage.value = 1
    loadPublishLogs()
  } else {
    adminPage.value = 1
    loadAdminLogs()
  }
}

const handleRefresh = () => {
  if (activeTab.value === 'publish') {
    loadPublishLogs()
  } else {
    loadAdminLogs()
  }
}

const formatDate = (date: string) => {
  return new Date(date).toLocaleString('zh-CN')
}

const getStatusType = (status: string) => {
  const map: Record<string, string> = {
    success: 'success',
    failed: 'danger',
    pending: 'warning',
  }
  return map[status] || 'info'
}

const getActionType = (action: string) => {
  const map: Record<string, string> = {
    delete_package: 'danger',
    create_user: 'success',
    reset_token: 'warning',
    restore_package: 'info',
    admin_login: 'primary',
  }
  return map[action] || ''
}

const formatAction = (action: string) => {
  const map: Record<string, string> = {
    delete_package: '删除包',
    create_user: '创建用户',
    reset_token: '重置Token',
    restore_package: '恢复包',
    hard_delete_package: '永久删除',
    toggle_user: '切换用户状态',
    admin_login: '管理员登录',
  }
  return map[action] || action
}

onMounted(() => {
  loadPublishLogs()
})
</script>

<template>
  <div class="logs-container">
    <div class="page-header">
      <h1>
        <el-icon :size="28"><Document /></el-icon>
        操作日志
      </h1>
      <el-button :icon="Refresh" @click="handleRefresh">刷新</el-button>
    </div>

    <!-- 筛选栏 -->
    <el-card class="search-card" shadow="hover">
      <el-row :gutter="20">
        <el-col :xs="24" :sm="12">
          <el-select
            v-model="selectedStatus"
            placeholder="筛选发布状态"
            clearable
            style="width: 100%"
            @change="handleSearch"
            :disabled="activeTab !== 'publish'"
          >
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-select
            v-model="selectedAction"
            placeholder="筛选操作类型"
            clearable
            style="width: 100%"
            @change="handleSearch"
            :disabled="activeTab !== 'admin'"
          >
            <el-option label="删除包" value="delete_package" />
            <el-option label="创建用户" value="create_user" />
            <el-option label="重置Token" value="reset_token" />
            <el-option label="恢复包" value="restore_package" />
            <el-option label="管理员登录" value="admin_login" />
          </el-select>
        </el-col>
      </el-row>
    </el-card>

    <!-- 日志表格 -->
    <el-card shadow="hover">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 发布日志 -->
        <el-tab-pane label="发布日志" name="publish">
          <el-table
            :data="publishLogs"
            v-loading="publishLoading"
            stripe
            style="width: 100%"
          >
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="package_name" label="包名" min-width="150" />
            <el-table-column prop="version" label="版本" width="100" />
            <el-table-column prop="organization" label="组织" width="120" />
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="getStatusType(row.status)" size="small">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="error" label="错误信息" min-width="200" show-overflow-tooltip>
              <template #default="{ row }">
                <span v-if="row.error" style="color: #f56c6c">{{ row.error }}</span>
                <span v-else style="color: #909399">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="ip_addr" label="IP地址" width="140" />
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">
                {{ formatDate(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 管理员操作日志 -->
        <el-tab-pane label="操作日志" name="admin">
          <el-table
            :data="adminLogs"
            v-loading="adminLoading"
            stripe
            style="width: 100%"
          >
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="action" label="操作" width="150">
              <template #default="{ row }">
                <el-tag v-if="getActionType(row.action)" :type="getActionType(row.action)" size="small">
                  {{ formatAction(row.action) }}
                </el-tag>
                <span v-else>{{ row.action }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="target" label="目标对象" min-width="150" show-overflow-tooltip />
            <el-table-column prop="ip_addr" label="IP地址" width="140" />
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">
                {{ formatDate(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<style scoped>
.logs-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
}

.page-header h1 {
  font-size: 28px;
  color: #303133;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-card {
  margin-bottom: 20px;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 15px;
    align-items: flex-start;
  }

  .page-header h1 {
    font-size: 22px;
  }
}
</style>
