# CJRepo 包管理 API 测试

# 运行方式:
#   hurl --variable admin_key=46ab829cae2d14886ec673dd2fea9f26 tests/package-management.hurl

# 1. 登录获取 Token
POST http://localhost:8060/api/admin/login
Content-Type: application/json
{
  "adminKey": "{{admin_key}}"
}
HTTP 200
[Captures]
token: jsonpath("$.token")

# 2. 获取 Dashboard 统计
GET http://localhost:8060/api/admin/dashboard
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.packages") >= 0
jsonpath("$.versions") >= 0
jsonpath("$.users") >= 0
jsonpath("$.storageSize") >= 0

# 3. 获取包列表（空或已有）
GET http://localhost:8060/api/admin/packages
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.total") >= 0
[Captures]
package_count: jsonpath("$.total")

# 4. 获取包列表（带分页）
GET http://localhost:8060/api/admin/packages?page=1&pageSize=10
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.page") == 1
jsonpath("$.pageSize") == 10

# 5. 获取包列表（搜索）
GET http://localhost:8060/api/admin/packages?search=test
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.total") >= 0

# 6. 获取组织列表（用于筛选）
GET http://localhost:8060/api/admin/organizations
Authorization: Bearer {{token}}
HTTP 200

# 7. 创建测试组织（用于后续测试）
POST http://localhost:8060/api/admin/organizations
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "name": "pkg-test-org",
  "display_name": "包测试组织"
}
HTTP 200
[Captures]
test_org_id: jsonpath("$.id")

# 8. 获取包列表（按组织筛选）
GET http://localhost:8060/api/admin/packages?org=pkg-test-org
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.total") == 0

# 9. 获取发布日志
GET http://localhost:8060/api/admin/logs/publish?page=1&pageSize=10
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.total") >= 0

# 10. 获取管理员操作日志
GET http://localhost:8060/api/admin/logs/admin?page=1&pageSize=10
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.total") >= 0

# 11. 清理测试组织
DELETE http://localhost:8060/api/admin/organizations/{{test_org_id}}
Authorization: Bearer {{token}}
HTTP 200