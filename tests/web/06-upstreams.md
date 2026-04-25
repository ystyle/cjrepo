# 上游管理

验证上游源的 CRUD 和连接测试功能。

## 前置条件

- 已登录进入管理后台

## TC-06-01: 页面加载

```bash
agent-browser open http://localhost:8060/admin/upstreams
sleep 2
agent-browser console --clear
agent-browser snapshot -i
```

**预期：**
- 页面标题"上游管理"
- `button "添加上游"`
- 表格列：名称、URL、缓存时间、启用、最后同步、操作
- `agent-browser console` 无报错

## TC-06-02: 添加上游源

```bash
agent-browser console --clear
agent-browser click <添加上游按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出"添加上游"对话框：
- `textbox "* 上游名称"`
- `textbox "* 上游地址"`（URL 格式）
- 缓存时间、认证令牌、启用开关等
- `button "确定"`、`button "取消"`

```bash
agent-browser fill <名称 ref> "test-upstream"
agent-browser fill <URL ref> "https://repo.example.com"
agent-browser click <确定按钮 ref>
sleep 2
```

**预期：** 对话框关闭，表格中出现新上游源

## TC-06-03: 连接测试

```bash
agent-browser click <test-upstream 行 测试按钮 ref>
sleep 3
```

**预期：** 显示测试结果提示（成功/失败）

## TC-06-04: 编辑上游

```bash
agent-browser click <test-upstream 行 编辑按钮 ref>
sleep 2
```

**预期：** 弹出编辑对话框，数据已回填

## TC-06-05: 删除上游

```bash
agent-browser click <test-upstream 行 删除按钮 ref>
sleep 2
agent-browser click <确认删除按钮 ref>
sleep 2
```

**预期：** 该上游从表格中移除
