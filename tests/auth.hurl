# CJRepo 认证 API 测试

# 运行方式:
#   export CJREPO_ADMIN_KEY=test-admin-key-2024
#   hurl --variable admin_key=$CJREPO_ADMIN_KEY tests/auth.hurl

# 1. 健康检查
GET http://localhost:8060/health
HTTP 200
[Asserts]
jsonpath("$.status") == "ok"

# 2. 公开统计信息
GET http://localhost:8060/api/stats
HTTP 200
[Asserts]
jsonpath("$.packages") >= 0
jsonpath("$.users") >= 0
jsonpath("$.versions") >= 0
jsonpath("$.siteName") exists

# 3. 登录 - 缺少密钥
POST http://localhost:8060/api/admin/login
Content-Type: application/json
{}
HTTP 400
[Asserts]
jsonpath("$.error") exists

# 4. 登录 - 无效密钥
POST http://localhost:8060/api/admin/login
Content-Type: application/json
{
  "adminKey": "invalid-key"
}
HTTP 401
[Asserts]
jsonpath("$.error") == "Invalid admin key"

# 5. 登录 - 正确密钥
POST http://localhost:8060/api/admin/login
Content-Type: application/json
{
  "adminKey": "{{admin_key}}"
}
HTTP 200
[Asserts]
jsonpath("$.token") exists
jsonpath("$.expiresAt") > 0
jsonpath("$.message") == "Login successful"
[Captures]
token: jsonpath("$.token")

# 6. 使用 Token 访问 Dashboard
GET http://localhost:8060/api/admin/dashboard
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.packages") >= 0
jsonpath("$.versions") >= 0
jsonpath("$.users") >= 0
jsonpath("$.activeUsers") >= 0
jsonpath("$.storageSize") >= 0
jsonpath("$.publishSuccess") >= 0
jsonpath("$.publishFailed") >= 0

# 7. 无 Token 访问管理 API
GET http://localhost:8060/api/admin/users
HTTP 401
[Asserts]
jsonpath("$.error") == "Authorization header is required"

# 8. 无效 Token 访问管理 API
GET http://localhost:8060/api/admin/users
Authorization: Bearer invalid-token
HTTP 401
[Asserts]
jsonpath("$.error") == "Invalid or expired token"

# 9. 错误格式 Token（缺少 Bearer）
GET http://localhost:8060/api/admin/users
Authorization: {{token}}
HTTP 401
[Asserts]
jsonpath("$.error") == "Invalid authorization format. Use: Bearer <token>"