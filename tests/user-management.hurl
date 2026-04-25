# CJRepo 用户管理 API 测试

# 运行方式:
#   hurl --variable admin_key=46ab829cae2d14886ec673dd2fea9f26 tests/user-management.hurl

# 1. 登录获取 Token
POST http://localhost:8060/api/admin/login
Content-Type: application/json
{
  "adminKey": "{{admin_key}}"
}
HTTP 200
[Captures]
token: jsonpath("$.token")

# 2. 获取用户列表
GET http://localhost:8060/api/admin/users
Authorization: Bearer {{token}}
HTTP 200
[Captures]
initial_count: jsonpath("$.length")

# 3. 创建用户
POST http://localhost:8060/api/admin/users
Authorization: Bearer {{token}}
Content-Type: application/json
{
  "username": "testuser",
  "email": "test@example.com"
}
HTTP 201
[Asserts]
jsonpath("$.username") == "testuser"
jsonpath("$.email") == "test@example.com"
jsonpath("$.is_active") == true
jsonpath("$.token") startsWith "token-"
[Captures]
user_id: jsonpath("$.id")
user_token: jsonpath("$.token")

# 4. 获取用户列表（验证新增）
GET http://localhost:8060/api/admin/users
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") == {{initial_count}} + 1

# 5. 禁用用户
PUT http://localhost:8060/api/admin/users/{{user_id}}/toggle
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.is_active") == false

# 6. 再次禁用（启用）
PUT http://localhost:8060/api/admin/users/{{user_id}}/toggle
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.is_active") == true

# 7. 重置用户 Token
POST http://localhost:8060/api/admin/users/{{user_id}}/reset-token
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.token") != "{{user_token}}"
[Captures]
new_token: jsonpath("$.token")

# 8. 删除用户
DELETE http://localhost:8060/api/admin/users/{{user_id}}
Authorization: Bearer {{token}}
HTTP 200

# 9. 验证用户已删除
GET http://localhost:8060/api/admin/users
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath("$.length") == {{initial_count}}