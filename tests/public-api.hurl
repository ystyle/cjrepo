# CJRepo 公开 API 测试（无需认证）

# 运行方式:
#   hurl tests/public-api.hurl

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
jsonpath("$.downloads") >= 0
jsonpath("$.siteName") exists

# 3. 公开包列表（分页）
GET http://localhost:8060/api/packages?page=1&pageSize=20
HTTP 200
[Asserts]
jsonpath("$.total") >= 0
jsonpath("$.page") == 1
jsonpath("$.pageSize") == 20

# 4. 公开包列表（搜索）
GET http://localhost:8060/api/packages?search=test
HTTP 200
[Asserts]
jsonpath("$.total") >= 0

# 5. 公开组织列表
GET http://localhost:8060/api/organizations
HTTP 200
[Asserts]
jsonpath("$") isArray