<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  ElCard,
  ElTable,
  ElButton,
  ElTag,
  ElEmpty,
  ElMessage,
  ElMessageBox,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElSelect,
  ElOption,
  ElIcon,
  ElTooltip,
  ElPopconfirm,
} from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  UserFilled,
  OfficeBuilding,
  Box,
  Key,
} from '@element-plus/icons-vue'
import {
  getTeams,
  createTeam,
  updateTeam,
  deleteTeam,
  getTeamOrganizations,
  updateTeamOrganizations,
  getTeamPackages,
  updateTeamPackages,
  getTeamMembers,
  updateTeamMembers,
  type Team,
  type TeamOrganization,
  type TeamPackage,
  type TeamMember,
} from '../../api/team'
import {
  getOrganizations,
  getUsers,
  type Organization,
  type User,
} from '../../api/admin'

const teams = ref<Team[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingId = ref<number | null>(null)

const orgsDialogVisible = ref(false)
const packagesDialogVisible = ref(false)
const membersDialogVisible = ref(false)
const currentTeam = ref<Team | null>(null)

const selectedOrgIds = ref<(number | null)[]>([])
const selectedUserIds = ref<number[]>([])
const teamMembers = ref<TeamMember[]>([])
const teamPackages = ref<TeamPackage[]>([])
const newPackages = ref<any[]>([])

const selectedMembers = ref<User[]>([])
const searchMemberKeyword = ref('')
const searchMemberResults = ref<User[]>([])
let searchMemberTimer: ReturnType<typeof setTimeout>

const formRef = ref()
const formData = ref({
  name: '',
  display_name: '',
  description: '',
  permission: 'read',
})
const formRules = {
  name: [{ required: true, message: '请输入团队标识', trigger: 'blur' }],
  permission: [{ required: true, message: '请选择默认权限', trigger: 'change' }],
}

const permissionOptions = [
  { value: 'read', label: '读取', desc: '可下载包和查看索引' },
  { value: 'write', label: '写入', desc: '可发布新版本' },
  { value: 'overwrite', label: '覆盖', desc: '可覆盖已存在的版本' },
]

const loadTeams = async () => {
  loading.value = true
  try {
    const data = await getTeams()
    teams.value = data || []
  } catch (error: any) {
    console.error('加载团队列表失败:', error)
    teams.value = []
    ElMessage.error(error.response?.data?.error || '加载团队列表失败')
  } finally {
    loading.value = false
  }
}

// 组织关联：搜索 → 添加 → 列表
const selectedOrgs = ref<Organization[]>([])
const searchOrgKeyword = ref('')
const searchOrgResults = ref<Organization[]>([])
let searchOrgTimer: ReturnType<typeof setTimeout>

const onSearchOrgInput = () => {
  clearTimeout(searchOrgTimer)
  searchOrgTimer = setTimeout(async () => {
    const kw = searchOrgKeyword.value.trim()
    if (!kw) { searchOrgResults.value = []; return }
    try {
      const data = await getOrganizations({ search: kw })
      searchOrgResults.value = data || []
    } catch { searchOrgResults.value = [] }
  }, 300)
}

const addOrg = (org: Organization) => {
  if (selectedOrgIds.value.includes(org.id)) return
  selectedOrgIds.value.push(org.id)
  selectedOrgs.value.push(org)
  searchOrgKeyword.value = ''
  searchOrgResults.value = []
}

const removeOrg = (idx: number) => {
  selectedOrgIds.value.splice(idx, 1)
  selectedOrgs.value.splice(idx, 1)
}

const openOrgsDialog = async (team: Team) => {
  currentTeam.value = team
  orgsDialogVisible.value = true
  selectedOrgIds.value = []
  selectedOrgs.value = []
  searchOrgResults.value = []
  const data = await getTeamOrganizations(team.id)
  selectedOrgIds.value = data?.map((o: TeamOrganization) => o.organization_id) || []
  selectedOrgs.value = data?.filter((o) => o.organization_id != null).map((o) => ({
    id: o.organization_id!,
    name: o.organization_name,
    display_name: o.organization_name,
    description: '',
    is_default: false,
    member_count: 0,
    package_count: 0,
    created_at: '',
    updated_at: '',
  })) || []
}

const saveOrgs = async () => {
  if (!currentTeam.value) return
  try {
    await updateTeamOrganizations(currentTeam.value.id, selectedOrgIds.value)
    ElMessage.success('组织关联更新成功')
    orgsDialogVisible.value = false
    await loadTeams()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '更新失败')
  }
}

const openPackagesDialog = async (team: Team) => {
  currentTeam.value = team
  packagesDialogVisible.value = true
  const data = await getTeamPackages(team.id)
  teamPackages.value = data || []
  newPackages.value = data?.map((p: TeamPackage) => ({
    organization: p.organization,
    package_name: p.package_name,
    permission: p.permission,
  })) || []
}

const searchPkgOrgs = async (keyword: string, row: any) => {
  if (!keyword) return
  try {
    const data = await getOrganizations({ search: keyword })
    row._orgOptions = data || []
  } catch {
    row._orgOptions = []
  }
}

const addPackageRow = () => {
  newPackages.value.push({
    organization: '',
    package_name: '',
    permission: 'read',
    _orgOptions: [],
  })
}

const removePackageRow = (index: number) => {
  newPackages.value.splice(index, 1)
}

const savePackages = async () => {
  if (!currentTeam.value) return
  const validPackages = newPackages.value.filter(
    (p) => p.package_name && p.permission
  )
  try {
    await updateTeamPackages(currentTeam.value.id, validPackages)
    ElMessage.success('包权限更新成功')
    packagesDialogVisible.value = false
    await loadTeams()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '更新失败')
  }
}

const openMembersDialog = async (team: Team) => {
  currentTeam.value = team
  membersDialogVisible.value = true
  selectedUserIds.value = []
  selectedMembers.value = []
  searchMemberResults.value = []
  searchMemberKeyword.value = ''
  const data = await getTeamMembers(team.id)
  teamMembers.value = data || []
  selectedUserIds.value = data?.map((m: TeamMember) => m.user_id) || []
  selectedMembers.value = data?.map((m: TeamMember) => ({
    id: m.user_id,
    username: m.username,
    email: m.email,
    is_active: m.is_active,
  })) || []
}

const saveMembers = async () => {
  if (!currentTeam.value) return
  try {
    await updateTeamMembers(currentTeam.value.id, selectedUserIds.value)
    ElMessage.success('团队成员更新成功')
    membersDialogVisible.value = false
    await loadTeams()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '更新失败')
  }
}

const openCreateDialog = () => {
  dialogMode.value = 'create'
  editingId.value = null
  formData.value = { name: '', display_name: '', description: '', permission: 'read' }
  dialogVisible.value = true
}

const openEditDialog = (team: Team) => {
  dialogMode.value = 'edit'
  editingId.value = team.id
  formData.value = {
    name: team.name,
    display_name: team.display_name || '',
    description: team.description || '',
    permission: team.permission || 'read',
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return
    try {
      if (dialogMode.value === 'create') {
        await createTeam(formData.value)
        ElMessage.success('团队创建成功')
      } else {
        await updateTeam(editingId.value!, formData.value)
        ElMessage.success('团队更新成功')
      }
      dialogVisible.value = false
      await loadTeams()
    } catch (error: any) {
      ElMessage.error(error.response?.data?.error || '操作失败')
    }
  })
}

const handleDelete = async (team: Team) => {
  try {
    await ElMessageBox.confirm(
      `确定删除团队 "${team.display_name || team.name}"？此操作不可恢复。`,
      '确认删除',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteTeam(team.id)
    ElMessage.success('团队已删除')
    await loadTeams()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || '删除失败')
    }
  }
}

const onSearchMemberInput = () => {
  clearTimeout(searchMemberTimer)
  searchMemberTimer = setTimeout(async () => {
    const kw = searchMemberKeyword.value.trim()
    if (!kw) { searchMemberResults.value = []; return }
    try {
      const data = await getUsers({ search: kw })
      searchMemberResults.value = data || []
    } catch { searchMemberResults.value = [] }
  }, 300)
}

const addMember = (user: User) => {
  if (selectedUserIds.value.includes(user.id)) return
  selectedUserIds.value.push(user.id)
  selectedMembers.value.push(user)
  searchMemberKeyword.value = ''
  searchMemberResults.value = []
}

const removeMember = (idx: number) => {
  selectedUserIds.value.splice(idx, 1)
  selectedMembers.value.splice(idx, 1)
}

const getPermissionLabel = (perm: string) => {
  const opt = permissionOptions.find((o) => o.value === perm)
  return opt?.label || perm
}

const getPermissionType = (perm: string) => {
  switch (perm) {
    case 'read':
      return ''
    case 'write':
      return 'success'
    case 'overwrite':
      return 'warning'
    default:
      return ''
  }
}

onMounted(() => {
  loadTeams()
})
</script>

<template>
  <div class="teams-page">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">团队管理</h2>
        <p class="page-description">
          管理团队权限，精细化控制用户对组织/包的访问权限。
        </p>
      </div>
      <div class="header-right">
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">
          添加团队
        </el-button>
      </div>
    </div>

    <el-card v-loading="loading" class="teams-card">
      <el-table :data="teams" stripe>
        <el-table-column prop="name" label="标识" width="150">
          <template #default="{ row }">
            <div class="team-name">
              <el-icon><Key /></el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="display_name" label="团队名称" width="180">
          <template #default="{ row }">
            <span class="team-display-name">{{ row.display_name || row.name }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="description" label="描述" min-width="200">
          <template #default="{ row }">
            <span class="team-description">{{ row.description || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="permission" label="默认权限" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getPermissionType(row.permission)" size="small">
              {{ getPermissionLabel(row.permission) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="member_count" label="成员数" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small">{{ row.member_count }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="org_count" label="组织数" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.org_count }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="package_count" label="包权限" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="success">{{ row.package_count }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="400" align="center" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :icon="UserFilled" @click="openMembersDialog(row)">
              成员
            </el-button>
            <el-button size="small" :icon="OfficeBuilding" @click="openOrgsDialog(row)">
              组织
            </el-button>
            <el-button size="small" :icon="Box" @click="openPackagesDialog(row)">
              包
            </el-button>
            <el-button size="small" :icon="Edit" @click="openEditDialog(row)">
              编辑
            </el-button>
            <el-button size="small" type="danger" :icon="Delete" @click="handleDelete(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && teams.length === 0" description="暂无团队">
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">
          创建第一个团队
        </el-button>
      </el-empty>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '添加团队' : '编辑团队'"
      width="600px"
    >
      <el-form ref="formRef" :model="formData" :rules="formRules" label-width="100px">
        <el-form-item label="团队标识" prop="name">
          <el-input
            v-model="formData.name"
            placeholder="例如: dev-team"
            :disabled="dialogMode === 'edit'"
            clearable
          />
          <div class="form-tip">唯一标识，创建后不可修改</div>
        </el-form-item>

        <el-form-item label="团队名称">
          <el-input
            v-model="formData.display_name"
            placeholder="例如: 开发团队"
            clearable
          />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="团队描述"
            clearable
          />
        </el-form-item>

        <el-form-item label="默认权限" prop="permission">
          <el-select v-model="formData.permission" placeholder="选择权限级别">
            <el-option
              v-for="opt in permissionOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            >
              <div class="permission-option">
                <span>{{ opt.label }}</span>
                <span class="permission-desc">{{ opt.desc }}</span>
              </div>
            </el-option>
          </el-select>
          <div class="form-tip">团队成员对关联组织/包的默认权限</div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="orgsDialogVisible"
      :title="`组织关联 - ${currentTeam?.display_name || currentTeam?.name || ''}`"
      width="700px"
    >
      <div class="dialog-content">
        <p class="dialog-tip">搜索并添加组织：</p>
        <div class="search-row">
          <el-input
            v-model="searchOrgKeyword"
            placeholder="输入组织名称搜索"
            clearable
            @input="onSearchOrgInput"
            @clear="searchOrgResults = []"
          />
          <div v-if="searchOrgResults.length > 0" class="search-results">
            <div
              v-for="org in searchOrgResults"
              :key="org.id"
              class="search-result-item"
              @click="addOrg(org)"
            >
              <span>{{ org.display_name || org.name }}</span>
              <el-tag size="small" type="info">{{ org.name }}</el-tag>
              <el-button size="small" type="primary" link>添加</el-button>
            </div>
          </div>
        </div>
        <p class="dialog-tip" style="margin-top: 16px">已关联的组织：</p>
        <el-table :data="selectedOrgs" stripe max-height="300">
          <el-table-column prop="display_name" label="组织名称" min-width="200" />
          <el-table-column label="操作" width="100" align="center">
            <template #default="{ $index }">
              <el-button size="small" type="danger" link @click="removeOrg($index)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="selectedOrgs.length === 0" description="暂未关联组织" :image-size="80" />
      </div>
      <template #footer>
        <el-button @click="orgsDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveOrgs">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="packagesDialogVisible"
      :title="`包权限 - ${currentTeam?.display_name || currentTeam?.name || ''}`"
      width="800px"
    >
      <div class="dialog-content">
        <p class="dialog-tip">配置团队对特定包的权限（覆盖默认权限）：</p>
        <el-button size="small" type="primary" @click="addPackageRow">
          添加包
        </el-button>
        <el-table :data="newPackages" stripe style="margin-top: 10px">
          <el-table-column prop="organization" label="组织" width="150">
            <template #default="{ row }">
              <el-select
                v-model="row.organization"
                placeholder="搜索组织"
                size="small"
                filterable
                remote
                reserve-keyword
                clearable
                :remote-method="(kw: string) => searchPkgOrgs(kw, row)"
              >
                <el-option value="" label="(无组织)" />
                <el-option
                  v-for="org in row._orgOptions || []"
                  :key="org.id"
                  :value="org.name"
                  :label="org.display_name || org.name"
                />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column prop="package_name" label="包名" width="200">
            <template #default="{ row }">
              <el-input v-model="row.package_name" placeholder="包名" size="small" />
            </template>
          </el-table-column>
          <el-table-column prop="permission" label="权限" width="120">
            <template #default="{ row }">
              <el-select v-model="row.permission" size="small">
                <el-option v-for="opt in permissionOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="80" align="center">
            <template #default="{ $index }">
              <el-button size="small" type="danger" :icon="Delete" @click="removePackageRow($index)" />
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <el-button @click="packagesDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="savePackages">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="membersDialogVisible"
      :title="`团队成员 - ${currentTeam?.display_name || currentTeam?.name || ''}`"
      width="700px"
    >
      <div class="dialog-content">
        <p class="dialog-tip">搜索并添加成员：</p>
        <div class="search-row">
          <el-input
            v-model="searchMemberKeyword"
            placeholder="输入用户名搜索"
            clearable
            @input="onSearchMemberInput"
            @clear="searchMemberResults = []"
          />
          <div v-if="searchMemberResults.length > 0" class="search-results">
            <div
              v-for="user in searchMemberResults"
              :key="user.id"
              class="search-result-item"
              @click="addMember(user)"
            >
              <span>{{ user.username }}</span>
              <span class="search-result-email">{{ user.email }}</span>
              <el-tag :type="user.is_active ? 'success' : 'danger'" size="small">
                {{ user.is_active ? '活跃' : '禁用' }}
              </el-tag>
              <el-button size="small" type="primary" link>添加</el-button>
            </div>
          </div>
        </div>
        <p class="dialog-tip" style="margin-top: 16px">已选成员：</p>
        <el-table :data="selectedMembers" stripe max-height="300">
          <el-table-column prop="username" label="用户名" width="150" />
          <el-table-column prop="email" label="邮箱" min-width="200" />
          <el-table-column prop="is_active" label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.is_active ? 'success' : 'danger'" size="small">
                {{ row.is_active ? '活跃' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" align="center">
            <template #default="{ $index }">
              <el-button size="small" type="danger" link @click="removeMember($index)">移除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="selectedMembers.length === 0" description="暂未选择成员" :image-size="80" />
      </div>
      <template #footer>
        <el-button @click="membersDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveMembers">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.teams-page {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.header-left {
  flex: 1;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 8px 0;
}

.page-description {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

.teams-card {
  margin-bottom: 20px;
}

.team-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  font-family: 'Courier New', monospace;
}

.team-display-name {
  font-weight: 500;
  color: #303133;
}

.team-description {
  color: #606266;
  font-size: 13px;
}

.form-tip {
  margin-top: 4px;
  font-size: 12px;
  color: #909399;
}

.permission-option {
  display: flex;
  flex-direction: column;
}

.permission-desc {
  font-size: 12px;
  color: #909399;
}

.dialog-content {
  padding: 10px 0;
}

.dialog-tip {
  font-size: 14px;
  color: #606266;
  margin-bottom: 16px;
}

.search-row {
  position: relative;
}

.search-results {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  z-index: 100;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.1);
  max-height: 240px;
  overflow-y: auto;
}

.search-result-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  transition: background 0.2s;
}

.search-result-item:hover {
  background: #f5f7fa;
}

.search-result-email {
  font-size: 12px;
  color: #909399;
  flex: 1;
}

.user-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-name {
  font-weight: 500;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-right {
    width: 100%;
  }

  .header-right .el-button {
    width: 100%;
  }
}
</style>
