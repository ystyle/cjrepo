# 团队管理

验证团队管理页面的 CRUD 和四个弹窗功能。

## 前置条件

- 已登录进入管理后台
- 测试数据：至少有一个用户（用于测试成员添加）

```bash
# 准备测试用户
curl -s -X POST http://localhost:8060/api/admin/users \
  -H "Authorization: Bearer $(curl -s -X POST http://localhost:8060/api/admin/login \
    -H 'Content-Type: application/json' \
    -d "{\"adminKey\":\"$ADMIN_KEY\"}" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@test.com"}'

# 准备测试组织
curl -s -X POST http://localhost:8060/api/admin/organizations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-org","display_name":"测试组织"}'
```

## TC-02-01: 页面加载

```bash
agent-browser open http://localhost:8060/admin/teams
sleep 2
agent-browser console --clear
agent-browser snapshot -i
```

**预期：**
- 页面标题"团队管理"
- 描述文字"管理团队权限，精细化控制用户对组织/包的访问权限"
- `button "添加团队"` 可见
- 表格列：标识、团队名称、描述、默认权限、成员数、组织数、包权限、操作
- 底部 `el-pagination` 分页组件：`"20/page"`、`Go to previous page`(disabled)、`page 1`、`Go to next page`(disabled)
- `agent-browser console` 无报错

## TC-02-02: 创建团队

```bash
agent-browser console --clear
agent-browser click <添加团队按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出"添加团队"对话框，包含：
- `textbox "* 团队标识"`（必填）
- `textbox "团队名称"`
- `textbox "描述"`
- `el-select "默认权限"`（读取/写入/覆盖）
- `button "确定"`、`button "取消"`

```bash
agent-browser fill <标识 input ref> "test-team"
agent-browser fill <名称 input ref> "测试团队"
agent-browser click <确定按钮 ref>
sleep 2
```

**预期：**
- 对话框关闭
- 表格中出现新行，标识"test-team"，团队名称"测试团队"
- `agent-browser console` 无报错

## TC-02-03: 编辑团队

```bash
agent-browser console --clear
# 找到 test-team 行的编辑按钮
agent-browser click <test-team 行 编辑按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出"编辑团队"对话框：
- `textbox "* 团队标识"` 显示 `test-team` 且为禁用状态
- `textbox "团队名称"` 可编辑
- `button "确定"`、`button "取消"`

```bash
agent-browser fill <团队名称 ref> "测试团队(已编辑)"
agent-browser click <确定按钮 ref>
sleep 2
```

**预期：** 对话框中该团队名称更新为"测试团队(已编辑)"

## TC-02-04: 成员弹窗——搜索并添加用户

```bash
agent-browser console --clear
agent-browser click <test-team 行 成员按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出"团队成员 - 测试团队"对话框：
- `textbox "输入用户名搜索"`
- 已选成员列表（若有之前添加的成员）
- `button "取消"`、`button "保存"`

```bash
agent-browser type <搜索输入框 ref> "test"
sleep 1
agent-browser snapshot -i
```

**预期：** 搜索下拉出现匹配的用户结果，每条包含：用户名、邮箱、状态标签、`button "添加"`

```bash
agent-browser click <第一个搜索结果 ref>
sleep 1
agent-browser snapshot -i
```

**预期：**
- 搜索下拉关闭
- 已选成员表格中出现该用户（用户名、邮箱、状态、`button "移除"`）

```bash
agent-browser click <保存按钮 ref>
sleep 2
```

**预期：** 
- 对话框关闭
- test-team 行的"成员数"更新为 1+

## TC-02-05: 组织弹窗——搜索并关联组织

```bash
agent-browser console --clear
agent-browser click <test-team 行 组织按钮 ref>
sleep 2
```

**预期：** 弹出"组织关联 - 测试团队"对话框：
- `textbox "输入组织名称搜索"`
- 已关联组织列表（可移除）
- `button "取消"`、`button "保存"`

```bash
agent-browser type <搜索输入框 ref> "test"
sleep 1
agent-browser snapshot -i
```

**预期：** 搜索下拉出现匹配的组织

```bash
agent-browser click <搜索结果 ref>
sleep 1
```

**预期：** 组织被添加到已关联列表

```bash
agent-browser click <保存按钮 ref>
sleep 2
```

**预期：** test-team 行的"组织数"更新为 1+

## TC-02-06: 包弹窗——搜索并关联包

```bash
agent-browser console --clear
agent-browser click <test-team 行 包按钮 ref>
sleep 2
```

**预期：** 弹出"包权限 - 测试团队"对话框：
- 提示文字"搜索并添加包（支持 org::keyword 格式搜索组织包）"
- `textbox "输入包名搜索，或使用 org:: 搜索组织下的包"`
- 已关联包列表（若有之前关联的包）
- 底部提示"权限继承自团队默认级别"
- `button "取消"`、`button "保存"`

```bash
# 搜索无组织包
agent-browser fill <搜索输入框 ref> "my"
sleep 1
agent-browser snapshot -i
```

**预期：** 搜索下拉出现匹配的包结果，每条显示包名（有组织的显示 org::pkgname）、描述、`button "添加"`，同名多版本只显示一次

```bash
agent-browser click <搜索结果 添加按钮 ref>
sleep 1
```

**预期：** 搜索下拉关闭，包被添加到已关联列表表格，每行显示 org::pkgname（或仅 pkgname），含 `button "移除"`

```bash
# 搜索组织包（org::keyword 格式）
agent-browser fill <搜索输入框 ref> "myorg::"
sleep 1
agent-browser snapshot -i
```

**预期：** 搜索下拉显示 myorg 组织下的所有包

```bash
# 添加后再保存
agent-browser click <搜索结果 添加按钮 ref>
sleep 1
agent-browser click <保存按钮 ref>
sleep 2
```

**预期：** test-team 行的"包权限"数更新为对应的数量

## TC-02-07: 删除团队

```bash
agent-browser console --clear
agent-browser click <test-team 行 删除按钮 ref>
sleep 2
```

**预期：** 弹出"确认删除"对话框：
- 提示"确定删除团队 "测试团队"？此操作不可恢复。"
- `button "确定删除"`、`button "取消"`

```bash
agent-browser click <确定删除按钮 ref>
sleep 2
```

**预期：** 
- 对话框关闭
- 表格中该团队被移除
- `agent-browser console` 无报错

## TC-02-08: 空状态

当没有任何团队时：

```bash
agent-browser snapshot -i
```

**预期：** 
- 显示空状态提示"暂无团队"
- `button "创建第一个团队"` 可见

## TC-02-09: 分页

分页组件在团队列表底部始终可见：

```bash
agent-browser snapshot -i
```

**预期：**
- `generic "20/page"` 可见（页大小选择器）
- `button "Go to previous page"`（第一页时 disabled）
- `listitem "page 1"` 等页码按钮
- `button "Go to next page"`（有更多页时可点击）

当团队数量超过一页时：

```bash
agent-browser click <页码 2 ref>
sleep 2
```

**预期：**
- 表格显示第二页数据
- 当前页码高亮为 page 2
