# 认证与登录

验证管理后台登录流程和路由守卫。

## 前置条件

- 服务器已启动，`CJREPO_ADMIN_KEY` 已设置
- `agent-browser close --all`（保证干净的浏览器状态）

## TC-00-01: 未登录访问管理后台 → 重定向到登录页

```bash
agent-browser open http://localhost:8060/admin/teams
```

**预期：** URL 跳转到 `http://localhost:8060/admin/login?redirect=/admin/teams`
**验证：** `agent-browser snapshot -i` 应包含：
- `textbox "请输入管理密钥"`（密钥输入框）
- `button "登录"`
- `button "返回首页"`

## TC-00-02: 空密钥登录 → 应提示错误

```bash
agent-browser console --clear
agent-browser fill <textbox ref> ""
agent-browser click <登录按钮 ref>
```

**预期：** `agent-browser console` 应包含后端返回的错误提示

## TC-00-03: 错误密钥登录 → 应提示错误

```bash
agent-browser fill <textbox ref> "wrong-key"
agent-browser click <登录按钮 ref>
```

**预期：** `agent-browser console` 应包含 `"error": "invalid key"` 或类似错误信息

## TC-00-04: 正确密钥登录 → 成功进入后台

```bash
agent-browser console --clear
agent-browser fill <textbox ref> "$ADMIN_KEY"
agent-browser click <登录按钮 ref>
sleep 3
agent-browser snapshot -i
```

**预期：**
- URL 跳转到 `/admin/dashboard`（或 redirect 参数指定的路径）
- `agent-browser snapshot -i` 应包含菜单项：仪表盘、包管理、用户管理、组织管理、团队管理、上游管理、操作日志
- `agent-browser console` 无报错

## TC-00-05: 直接访问登录页（已登录）→ 重定向到仪表盘

```bash
agent-browser open http://localhost:8060/admin/login
sleep 2
```

**预期：** URL 自动跳转到 `/admin/dashboard`（路由守卫跳过登录页）
