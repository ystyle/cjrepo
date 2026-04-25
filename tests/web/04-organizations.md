# 组织管理

验证组织管理页面的 CRUD 功能。

## 前置条件

- 已登录进入管理后台

## TC-04-01: 页面加载

```bash
agent-browser open http://localhost:8060/admin/organizations
sleep 2
agent-browser console --clear
agent-browser snapshot -i
```

**预期：**
- 页面标题"组织管理"
- `button "添加组织"`
- 提示信息"权限通过团队管理配置"
- 表格列：标识、组织名称、描述、成员数、包数、默认、操作
- `agent-browser console` 无报错

## TC-04-02: 创建组织

```bash
agent-browser console --clear
agent-browser click <添加组织按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出"添加组织"对话框：
- `textbox "* 组织标识"`（必填）
- `textbox "组织名称"`
- `textbox "描述"`
- `button "确定"`、`button "取消"`

```bash
agent-browser fill <标识 ref> "test-org"
agent-browser fill <名称 ref> "测试组织"
agent-browser click <确定按钮 ref>
sleep 2
```

**预期：** 对话框关闭，表格中出现"test-org"

## TC-04-03: 编辑组织

```bash
agent-browser click <test-org 行 编辑按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出"编辑组织"对话框，标识为禁用状态

```bash
agent-browser fill <名称 ref> "测试组织(已编辑)"
agent-browser click <确定按钮 ref>
sleep 2
```

**预期：** 组织名称更新

## TC-04-04: 设为默认组织

```bash
agent-browser click <test-org 行 默认开关 ref>
sleep 2
```

**预期：** 开关状态切换，该组织被标记为默认
- 如果已经有默认组织，之前的默认组织自动取消

## TC-04-05: 删除组织

```bash
agent-browser click <test-org 行 删除按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出确认删除对话框

```bash
agent-browser click <确定删除按钮 ref>
sleep 2
```

**预期：** 组织从表格中移除
