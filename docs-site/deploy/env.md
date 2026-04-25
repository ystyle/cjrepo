# 环境变量

## 完整列表

| 变量 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `CJREPO_ADMIN_KEY` | ✅ | — | 管理后台登录密钥 |
| `CJREPO_PORT` | 否 | `8060` | HTTP 监听端口 |
| `CJREPO_REQUIRE_AUTH` | 否 | `false` | 设为 `true` 时下载/索引也需 Token 认证 |
| `CJREPO_DEFAULT_ORGANIZATION` | 否 | — | 设置后，所有发布请求不带 organization 参数时自动归属此组织 |
| `CJREPO_SITE_NAME` | 否 | `CJRepo` | 前端页面显示的站点名称 |
