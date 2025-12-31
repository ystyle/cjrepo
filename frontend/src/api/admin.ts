import request from './index'

// 登录请求
export interface LoginRequest {
  adminKey: string
}

// 登录响应
export interface LoginResponse {
  token: string
  expiresAt: number
  message: string
}

// 管理员登录
export const login = (data: LoginRequest) => {
  return request<LoginResponse>({
    url: '/admin/login',
    method: 'post',
    data,
  })
}

// Dashboard 统计
export interface DashboardStats {
  packages: number
  versions: number
  users: number
  activeUsers: number
  storageSize: number
  publishSuccess: number
  publishFailed: number
}

// 包列表响应
export interface PackagesResponse {
  data: Package[]
  total: number
  page: number
  pageSize: number
}

// 管理员查看的包信息
export interface Package {
  id: number
  organization: string
  name: string
  version: string
  description: string
  artifact_type: string
  executable: boolean
  authors: string
  repository: string
  homepage: string
  documentation: string
  tags: string
  categories: string
  licenses: string
  meta_version: number
  meta_data: string
  tarball_path: string
  tarball_size: number
  tarball_sha256: string
  created_at: string
  updated_at: string
  deleted_at: string
}

// 用户信息
export interface User {
  id: number
  username: string
  token: string
  email: string
  is_active: boolean
  created_at: string
}

// 创建用户请求
export interface CreateUserRequest {
  username: string
  email: string
}

// 重置 token 响应
export interface ResetTokenResponse {
  message: string
  token: string
}

// 获取 Dashboard 统计
export const getDashboardStats = () => {
  return request<DashboardStats>({
    url: '/admin/dashboard',
    method: 'get',
  })
}

// 获取所有包（管理员）
export const getAdminPackages = (params?: {
  page?: number
  pageSize?: number
  search?: string
  org?: string
  artifactType?: string
  deleted?: boolean
}) => {
  return request<PackagesResponse>({
    url: '/admin/packages',
    method: 'get',
    params,
  })
}

// 删除包
export const deletePackage = (id: number) => {
  return request({
    url: `/admin/packages/${id}`,
    method: 'delete',
  })
}

// 获取包的所有版本
export const getPackageVersions = (name: string, org?: string) => {
  return request<Package[]>({
    url: `/admin/packages/versions/${name}`,
    method: 'get',
    params: org ? { org } : undefined,
  })
}

// 恢复包
export const restorePackage = (id: number) => {
  return request({
    url: `/admin/packages/${id}/restore`,
    method: 'put',
  })
}

// 硬删除包
export const hardDeletePackage = (id: number) => {
  return request({
    url: `/admin/packages/${id}/hard`,
    method: 'delete',
  })
}

// 获取所有用户
export const getUsers = () => {
  return request<User[]>({
    url: '/admin/users',
    method: 'get',
  })
}

// 创建用户
export const createUser = (data: CreateUserRequest) => {
  return request<User>({
    url: '/admin/users',
    method: 'post',
    data,
  })
}

// 删除用户
export const deleteUser = (id: number) => {
  return request({
    url: `/admin/users/${id}`,
    method: 'delete',
  })
}

// 启用/禁用用户
export const toggleUser = (id: number) => {
  return request<User>({
    url: `/admin/users/${id}/toggle`,
    method: 'put',
  })
}

// 重置用户 token
export const resetUserToken = (userId: number) => {
  return request<ResetTokenResponse>({
    url: `/admin/users/${userId}/reset-token`,
    method: 'post',
  })
}

// 发布日志
export interface PublishLog {
  id: number
  package_name: string
  version: string
  organization: string
  status: string
  error: string
  ip_addr: string
  user_agent: string
  created_at: string
}

// 发布日志响应
export interface PublishLogsResponse {
  data: PublishLog[]
  total: number
  page: number
  pageSize: number
}

// 获取发布日志
export const getPublishLogs = (params?: {
  page?: number
  pageSize?: number
  status?: string
}) => {
  return request<PublishLogsResponse>({
    url: '/admin/logs/publish',
    method: 'get',
    params,
  })
}

// 管理员操作日志
export interface AdminLog {
  id: number
  action: string
  target: string
  details: string
  ip_addr: string
  user_agent: string
  created_at: string
}

// 管理员操作日志响应
export interface AdminLogsResponse {
  data: AdminLog[]
  total: number
  page: number
  pageSize: number
}

// 获取管理员操作日志
export const getAdminLogs = (params?: {
  page?: number
  pageSize?: number
  action?: string
}) => {
  return request<AdminLogsResponse>({
    url: '/admin/logs/admin',
    method: 'get',
    params,
  })
}

// 上游配置
export interface Upstream {
  id: number
  name: string
  url: string
  enabled: boolean
  cache_ttl: number
  auth_token: string
  last_sync_at: string
  created_at: string
  updated_at: string
}

// 创建上游请求
export interface CreateUpstreamRequest {
  name: string
  url: string
  enabled?: boolean
  cache_ttl?: number
  auth_token?: string
}

// 更新上游请求
export interface UpdateUpstreamRequest {
  url?: string
  enabled?: boolean
  cache_ttl?: number
  auth_token?: string
}

// 获取上游列表
export const getUpstreams = () => {
  return request<Upstream[]>({
    url: '/admin/upstreams',
    method: 'get',
  })
}

// 创建上游
export const createUpstream = (data: CreateUpstreamRequest) => {
  return request<Upstream>({
    url: '/admin/upstreams',
    method: 'post',
    data,
  })
}

// 更新上游
export const updateUpstream = (id: number, data: UpdateUpstreamRequest) => {
  return request<Upstream>({
    url: `/admin/upstreams/${id}`,
    method: 'put',
    data,
  })
}

// 删除上游
export const deleteUpstream = (id: number) => {
  return request({
    url: `/admin/upstreams/${id}`,
    method: 'delete',
  })
}

// 测试上游连接
export const testUpstream = (id: number) => {
  return request<{ success: boolean; message: string }>({
    url: `/admin/upstreams/${id}/test`,
    method: 'post',
  })
}


