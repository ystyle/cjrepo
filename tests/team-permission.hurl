# CJRepo 团队权限 API 测试
# 使用 hurl 进行 API 测试
#
# 运行方式:
#   export CJREPO_ADMIN_KEY=test-admin-key-2024
#   hurl --variable admin_key=$CJREPO_ADMIN_KEY tests/team-permission.hurl

# 1. 健康检查
GET http://localhost:8060/health
HTTP 200
[Asserts]
json.path("$.status") == "ok"

# 2. 登录获取 JWT Token
POST http://localhost:8060/api/admin/login
Content-Type: application/json
{
  "adminKey": "{{admin_key}}"
}
HTTP 200
[Asserts]
json.path("$.token") exists
json.path("$.message") == "Login successful"
[Captures]
token: jsonpath("$.token")

# 3. 获取团队列表（空）
GET http://localhost:8060/api/admin/teams
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$") == []

# 4. 创建团队
POST http://localhost:8060/api/admin/teams
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "name": "dev-team",
  "display_name": "开发团队",
  "description": "开发人员团队",
  "permission": "write"
}
HTTP 200
[Asserts]
jsonpath("$.name") == "dev-team"
jsonpath("$.display_name") == "开发团队"
jsonpath("$.permission") == "write"
[Captures]
team_id: jsonpath("$.id")

# 5. 获取团队列表（1个团队）
GET http://localhost:8060/api/admin/teams
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") == 1
jsonpath("$[0].name") == "dev-team"
jsonpath("$[0].permission") == "write"
jsonpath("$[0].member_count") == 0
jsonpath("$[0].org_count") == 0
jsonpath("$[0].package_count") == 0

# 6. 更新团队
PUT http://localhost:8060/api/admin/teams/{{team_id}}
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "display_name": "开发团队（已更新）",
  "permission": "overwrite"
}
HTTP 200
[Asserts]
jsonpath("$.display_name") == "开发团队（已更新）"
jsonpath("$.permission") == "overwrite"

# 7. 创建用户（用于测试团队成员）
POST http://localhost:8060/api/admin/users
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "username": "testuser",
  "email": "test@example.com"
}
HTTP 201
[Captures]
user_id: jsonpath("$.id")
user_token: jsonpath("$.token")

# 8. 添加团队成员
PUT http://localhost:8060/api/admin/teams/{{team_id}}/members
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "user_ids": [{{user_id}}]
}
HTTP 200
[Asserts]
jsonpath("$.length") == 1
jsonpath("$[0].user_id") == {{user_id}}
jsonpath("$[0].username") == "testuser"

# 9. 获取团队成员列表
GET http://localhost:8060/api/admin/teams/{{team_id}}/members
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") == 1
jsonpath("$[0].username") == "testuser"

# 10. 创建组织（用于测试团队组织关联）
POST http://localhost:8060/api/admin/organizations
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "name": "test-org",
  "display_name": "测试组织",
  "description": "测试用组织"
}
HTTP 200
[Captures]
org_id: jsonpath("$.id")

# 11. 添加团队组织关联
PUT http://localhost:8060/api/admin/teams/{{team_id}}/organizations
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "organization_ids": [{{org_id}}]
}
HTTP 200
[Asserts]
jsonpath("$.length") == 1
jsonpath("$[0].organization_id") == {{org_id}}

# 12. 获取团队组织列表
GET http://localhost:8060/api/admin/teams/{{team_id}}/organizations
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") == 1
jsonpath("$[0].organization_name") == "test-org"

# 13. 添加团队包关联（权限走 team.permission）
PUT http://localhost:8060/api/admin/teams/{{team_id}}/packages
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "packages": [
    {
      "organization": "test-org",
      "package_name": "test-package"
    }
  ]
}
HTTP 200
[Asserts]
jsonpath("$.length") == 1
jsonpath("$[0].organization") == "test-org"
jsonpath("$[0].package_name") == "test-package"

# 14. 获取团队包关联列表
GET http://localhost:8060/api/admin/teams/{{team_id}}/packages
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") == 1
jsonpath("$[0].package_name") == "test-package"

# 15. 检查团队列表统计更新
GET http://localhost:8060/api/admin/teams
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$[0].member_count") == 1
jsonpath("$[0].org_count") == 1
jsonpath("$[0].package_count") == 1

# 16. 清空团队成员
PUT http://localhost:8060/api/admin/teams/{{team_id}}/members
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "user_ids": []
}
HTTP 200
[Asserts]
jsonpath("$.length") == 0

# 17. 清空团队组织
PUT http://localhost:8060/api/admin/teams/{{team_id}}/organizations
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "organization_ids": []
}
HTTP 200
[Asserts]
jsonpath("$.length") == 0

# 18. 清空团队包关联
PUT http://localhost:8060/api/admin/teams/{{team_id}}/packages
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "packages": []
}
HTTP 200
[Asserts]
jsonpath("$.length") == 0

# 19. 删除团队
DELETE http://localhost:8060/api/admin/teams/{{team_id}}
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.message") == "team deleted"

# 20. 验证团队已删除
GET http://localhost:8060/api/admin/teams
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") == 0

# 21. 清理测试数据：删除用户
DELETE http://localhost:8060/api/admin/users/{{user_id}}
Authorization: Bearer {{token}}
HTTP 200

# 22. 清理测试数据：删除组织
DELETE http://localhost:8060/api/admin/organizations/{{org_id}}
Authorization: Bearer {{token}}
HTTP 200