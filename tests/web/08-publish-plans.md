# 发布计划

验证发布计划页面的列表、创建和详情功能。

## 前置条件

- 已登录进入管理后台
- 测试数据：至少有一个已发布的包（用于创建计划时选择）

```bash
# 准备测试包（确保 ID=1 存在）
curl -s -X POST http://localhost:8060/api/admin/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"username":"testpub","email":"pub@test.com"}'

TOKEN=$(curl -s ...)
# 发布一个测试包
```

## TC-08-01: 列表页加载

```bash
agent-browser open http://localhost:8060/admin/publish-plans
sleep 3
agent-browser console --clear
agent-browser snapshot -i
```

**预期：**
- 页面标题"发布计划"
- 描述文字"批量发布包到目标上游仓库"
- `button "新建计划"` 可见
- 底部分页组件可见（空列表时显示空状态）
- `agent-browser console` 无报错

## TC-08-02: 创建计划——选择包

```bash
agent-browser console --clear
agent-browser click <新建按钮 ref>
sleep 3
agent-browser snapshot -i
```

**预期：**
- 页面标题"新建发布计划"
- 三步步骤条可见（选择包 → 分析结果 → 确认创建）
- 当前高亮步骤为"选择包"
- `el-select "目标上游"` 可见
- `textbox` 搜索包输入框可见
- `button "分析依赖"` 可见
- `agent-browser console` 无报错

```bash
# 选择目标上游
agent-browser click <目标上游选择器 ref>
sleep 1
agent-browser click <官方中心仓选项 ref>
sleep 1

# 搜索并添加包
agent-browser fill <搜索包输入框 ref> "mypkg"
sleep 1
agent-browser click <搜索结果 添加按钮 ref>
sleep 1
agent-browser snapshot -i
```

**预期：**
- 包标签出现在已选列表中
- 已选列表不为空

## TC-08-03: 分析依赖

```bash
agent-browser console --clear
agent-browser click <分析依赖按钮 ref>
sleep 3
agent-browser snapshot -i
```

**预期：**
- 成功后自动跳到步骤 2（分析结果）
- 步骤条高亮第 2 步
- 显示分析结果列表，按分类分组
- 每项包含复选框、包名、版本
- 已存在组不可勾选
- `agent-browser console` 无报错

```bash
# 勾选/取消勾选
agent-browser click <第一个包的勾选框 ref>
sleep 1
agent-browser click <下一步按钮 ref>
sleep 1
agent-browser snapshot -i
```

**预期：**
- 跳到步骤 3（确认创建）
- 显示已选包列表表格
- 显示计划名称输入框

## TC-08-04: 创建并查看计划

```bash
agent-browser console --clear
agent-browser fill <计划名称输入框 ref> "测试发布 v1"
sleep 1
agent-browser click <创建计划按钮 ref>
sleep 3
```

**预期：**
- 跳转到列表页
- 表格中出现新行，名称为"测试发布 v1"
- 状态为"等待中"
- `agent-browser console` 无报错

```bash
agent-browser click <新行 查看按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：**
- 进入详情页
- 顶部显示计划基本信息卡片（状态、目标上游、创建时间）
- 进度条可见
- 发布项列表显示创建的包
- `button "开始"` 可见
- `button "删除"` 可见

## TC-08-05: 开始执行

```bash
agent-browser console --clear
agent-browser click <开始按钮 ref>
sleep 2
```

**预期：**
- 控制台无报错
- `button "暂停"` 出现（开始后变为暂停按钮）
- `button "开始"` 消失

## TC-08-06: 返回列表验证状态

```bash
agent-browser click <返回列表 ref>
sleep 2
agent-browser snapshot -i
```

**预期：**
- 列表页该计划状态更新为"运行中"或"已完成"
- `agent-browser console` 无报错

## TC-08-07: 删除计划

```bash
agent-browser console --clear
agent-browser click <删除按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：**
- 弹出确认对话框
- `button "确定删除"`、`button "取消"`

```bash
agent-browser click <确定删除 ref>
sleep 2
```

**预期：**
- 对话框关闭
- 表格中该计划被移除
- `agent-browser console` 无报错

## TC-08-08: 空状态

当没有任何发布计划时：

```bash
agent-browser snapshot -i
```

**预期：**
- 显示空状态提示"暂无发布计划"
- `button "创建第一个计划"` 可见
