# CJRepo 组织管理 API 测试

# 运行方式:
#   hurl --variable admin_key=46ab829cae2d14886ec673dd2fea9f26 tests/organization.hurl

# 1. 登录获取 Token
POST http://localhost:8060/api/admin/login
Content-Type: application/json
{
  "adminKey": "{{admin_key}}"
}
HTTP 200
[Captures]
token: jsonpath("$.token")

# 2. 获取组织列表
GET http://localhost:8060/api/admin/organizations
Authorization: Bearer {{token}}
HTTP 200

# 3. 创建组织
POST http://localhost:8060/api/admin/organizations
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "name": "test-org",
  "display_name": "测试组织",
  "description": "测试用组织"
}
HTTP 200
[Asserts]
jsonpath("$.name") == "test-org"
jsonpath("$.display_name") == "测试组织"
jsonpath("$.is_default") == false
[Captures]
org_id: jsonpath("$.id")

# 4. 更新组织
PUT http://localhost:8060/api/admin/organizations/{{org_id}}
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "display_name": "测试组织（已更新）",
  "description": "更新后的描述"
}
HTTP 200
[Asserts]
jsonpath("$.display_name") == "测试组织（已更新）"
jsonpath("$.description") == "更新后的描述"

# 5. 创建用户用于成员测试
POST http://localhost:8060/api/admin/users
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "username": "orgmember",
  "email": "orgmember@example.com"
}
HTTP 201
[Captures]
member_user_id: jsonpath("$.id")

# 6. 添加组织成员
POST http://localhost:8060/api/admin/organizations/{{org_id}}/members
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "username": "orgmember"
}
HTTP 200
[Asserts]
jsonpath("$.message") == "member added successfully"

# 7. 获取组织成员列表
GET http://localhost:8060/api/admin/organizations/{{org_id}}/members
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") >= 1
jsonpath("$[0].username") == "orgmember"

# 8. 移除组织成员
DELETE http://localhost:8060/api/admin/organizations/{{org_id}}/members/{{member_user_id}}
Authorization: Bearer {{token}}
HTTP 200

# 9. 验证成员已移除
GET http://localhost:8060/api/admin/organizations/{{org_id}}/members
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") == 0

# 10. 删除组织
DELETE http://localhost:8060/api/admin/organizations/{{org_id}}
Authorization: Bearer {{token}}
HTTP 200

# 11. 清理测试用户
DELETE http://localhost:8060/api/admin/users/{{member_user_id}}
Authorization: Bearer {{token}}
HTTP 200