# 用户管理

验证用户管理页面的 CRUD 和 Token 功能。

## 前置条件

- 已登录进入管理后台

## TC-03-01: 页面加载

```bash
agent-browser open http://localhost:8060/admin/users
sleep 2
agent-browser console --clear
agent-browser snapshot -i
```

**预期：**
- 页面标题"用户管理"
- `button "创建用户"`
- 表格列：ID、用户名、邮箱、状态、操作
- `agent-browser console` 无报错

## TC-03-02: 创建用户

```bash
agent-browser console --clear
agent-browser click <创建用户按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出"创建用户"对话框：
- `textbox "用户名"`
- `textbox "邮箱"`
- 所属组织下拉（可选）
- `button "确定"`、`button "取消"`

```bash
agent-browser fill <用户名 ref> "newuser"
agent-browser fill <邮箱 ref> "newuser@test.com"
agent-browser click <确定按钮 ref>
sleep 2
```

**预期：**
- 关闭对话框（或弹出 Token 显示对话框）
- 如弹出 Token 对话框：显示 `textarea` 含 Token 字符串，`button "复制 Token"`
- 表格中出现新用户，用户名"newuser"

## TC-03-03: 复制 Token

```bash
# 若创建用户后自动弹出 Token 对话框
sleep 1
agent-browser snapshot -i
```

**预期：** 对话框提示"请妥善保管此 Token，关闭后将无法再次查看"

```bash
agent-browser click <复制 Token 按钮 ref>
```

**预期：** Token 被复制到剪贴板

## TC-03-04: 搜索用户

```bash
agent-browser type <搜索框 ref> "newuser"
sleep 1
```

**预期：** 表格仅显示匹配"newuser"的用户

```bash
# 清除搜索
agent-browser click <搜索框清除按钮 ref>
sleep 1
```

**预期：** 表格恢复显示所有用户

## TC-03-05: 重置 Token

```bash
agent-browser console --clear
agent-browser click <newuser 行 重置Token按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出新 Token 对话框，Token 已变更

## TC-03-06: 启用/禁用用户

```bash
agent-browser click <newuser 行 开关 ref>
sleep 2
```

**预期：** 用户状态在活跃/禁用之间切换
- `agent-browser console` 无报错

## TC-03-07: 删除用户

```bash
agent-browser console --clear
agent-browser click <newuser 行 删除按钮 ref>
sleep 2
```

**预期：** 弹出确认删除对话框

```bash
agent-browser click <确认删除按钮 ref>
sleep 2
```

**预期：** 该用户从表格中移除
