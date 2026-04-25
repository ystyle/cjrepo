# 包管理

验证包管理页面的搜索/筛选/查看版本功能。

## 前置条件

- 已登录进入管理后台
- 至少有一个已发布的包（可通过 cjpm publish 或 hurl 测试创建）

## TC-05-01: 页面加载

```bash
agent-browser open http://localhost:8060/admin/packages
sleep 2
agent-browser console --clear
agent-browser snapshot -i
```

**预期：**
- 页面标题"包管理"
- 搜索框、组织筛选、类型筛选
- 表格列：ID、包名、最新版本、组织、类型、描述、大小、创建时间、操作
- `agent-browser console` 无报错

## TC-05-02: 搜索包

```bash
agent-browser type <搜索框 ref> "test"
sleep 1
```

**预期：** 表格按包名或描述过滤，显示包含"test"的包

```bash
# 清除搜索
agent-browser click <搜索框清除按钮 ref>
sleep 1
```

**预期：** 表格恢复显示所有包

## TC-05-03: 按组织筛选

```bash
agent-browser click <组织筛选下拉 ref>
sleep 1
agent-browser click <下拉选项 ref>
sleep 1
```

**预期：** 表格按选择的组织过滤

## TC-05-04: 查看版本列表

```bash
agent-browser click <某包行 版本按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出版本列表对话框，显示该包的所有历史版本

```bash
agent-browser click <关闭对话框按钮 ref>
```

**预期：** 对话框关闭

## TC-05-05: 删除包

```bash
agent-browser click <某包行 删除按钮 ref>
sleep 2
```

**预期：** 弹出确认删除对话框，要求输入包名确认
