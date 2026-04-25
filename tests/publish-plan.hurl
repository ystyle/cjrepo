# CJRepo 发布计划 API 测试
# 运行:
#   hurl --variable admin_key=$CJREPO_ADMIN_KEY tests/publish-plan.hurl

GET http://localhost:8060/health
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"

POST http://localhost:8060/api/admin/login
Content-Type: application/json
{"adminKey": "{{admin_key}}"}
HTTP 200
[Asserts]
jsonpath "$.token" exists
jsonpath "$.message" == "Login successful"
[Captures]
token: jsonpath "$.token"

GET http://localhost:8060/api/admin/publish-plans
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.page" == 1
jsonpath "$.pageSize" == 20

GET http://localhost:8060/api/admin/packages?pageSize=3
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.data" count >= 1
[Captures]
pkg_id: jsonpath "$.data[0].id"
pkg_name: jsonpath "$.data[0].name"

POST http://localhost:8060/api/admin/publish-plans/analyze
Authorization: Bearer {{token}}
Content-Type: application/json
{"package_ids": [{{pkg_id}}], "target_upstream": 2}
HTTP 200
[Asserts]
jsonpath "$.total" >= 1

POST http://localhost:8060/api/admin/publish-plans
Authorization: Bearer {{token}}
Content-Type: application/json
{"name": "hurl-test-plan", "target_upstream": 2, "package_ids": [{{pkg_id}}], "poll_interval": 60, "poll_timeout": 600}
HTTP 200
[Asserts]
jsonpath "$.name" == "hurl-test-plan"
jsonpath "$.status" == "pending"
jsonpath "$.total_count" == 1
jsonpath "$.poll_interval" == 60
jsonpath "$.id" exists
[Captures]
plan_id: jsonpath "$.id"

GET http://localhost:8060/api/admin/publish-plans/{{plan_id}}
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.plan.id" == {{plan_id}}
jsonpath "$.plan.name" == "hurl-test-plan"
jsonpath "$.plan.status" == "pending"
jsonpath "$.items" count == 1
jsonpath "$.items[0].status" == "pending"

# 验证列表中有刚创建的计划（不假定位置，仅验证存在）
GET http://localhost:8060/api/admin/publish-plans
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.total" >= 1

POST http://localhost:8060/api/admin/publish-plans/{{plan_id}}/start
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.status" == "started"

POST http://localhost:8060/api/admin/publish-plans/{{plan_id}}/pause
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.status" == "paused"

POST http://localhost:8060/api/admin/publish-plans/{{plan_id}}/resume
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.status" == "resumed"

DELETE http://localhost:8060/api/admin/publish-plans/{{plan_id}}
Authorization: Bearer {{token}}
HTTP 200
[Asserts]
jsonpath "$.status" == "deleted"

GET http://localhost:8060/api/admin/publish-plans/{{plan_id}}
Authorization: Bearer {{token}}
HTTP 404
