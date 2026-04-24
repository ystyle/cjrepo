# CJRepo 后端分页 API 测试
# 运行: hurl --test --variable base_url=http://localhost:8060 tests/hurl/pagination.hurl

# 1. 登录
POST {{base_url}}/api/admin/login
Content-Type: application/json
{
  "adminKey": "46ab829cae2d14886ec673dd2fea9f26"
}
HTTP 200
[Asserts]
jsonpath "$.token" exists
[Captures]
token: jsonpath "$.token"

# 2. 用户列表
GET {{base_url}}/api/admin/users?page=1&pageSize=5
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" exists
jsonpath "$.total" exists
jsonpath "$.page" == 1
jsonpath "$.pageSize" == 5

# 3. 用户列表第二页
GET {{base_url}}/api/admin/users?page=2&pageSize=5
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.page" == 2

# 4. 组织列表
GET {{base_url}}/api/admin/organizations?page=1&pageSize=10
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" exists
jsonpath "$.total" exists
jsonpath "$.page" == 1
jsonpath "$.pageSize" == 10

# 5. 上游列表
GET {{base_url}}/api/admin/upstreams?page=1&pageSize=10
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" exists
jsonpath "$.total" exists
jsonpath "$.page" == 1

# 6. 包列表
GET {{base_url}}/api/admin/packages?page=1&pageSize=10
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" exists
jsonpath "$.total" exists
jsonpath "$.page" == 1
jsonpath "$.pageSize" == 10

# 7. 发布日志
GET {{base_url}}/api/admin/logs/publish?page=1&pageSize=10
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" exists
jsonpath "$.total" exists
jsonpath "$.page" == 1

# 8. 管理员日志
GET {{base_url}}/api/admin/logs/admin?page=1&pageSize=10
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" exists
jsonpath "$.total" exists
jsonpath "$.page" == 1