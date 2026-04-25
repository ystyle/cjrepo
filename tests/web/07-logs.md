# 操作日志

验证日志页面的查看/筛选/清理功能。

## 前置条件

- 已登录进入管理后台
- 至少有一次发布记录或管理员操作记录

## TC-07-01: 页面加载

```bash
agent-browser open http://localhost:8060/admin/logs
sleep 2
agent-browser console --clear
agent-browser snapshot -i
```

**预期：**
- 页面标题"操作日志"
- 两个标签页："发布日志"、"管理员操作日志"
- `button "清理日志"`
- 发布日志表格列：ID、包名、版本、组织、状态、错误信息、IP、时间
- `agent-browser console` 无报错

## TC-07-02: 切换到管理员日志

```bash
agent-browser click <管理员操作日志标签页 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 显示管理员操作日志表格（操作类型、目标对象、IP、时间等）

## TC-07-03: 按状态筛选发布日志

```bash
agent-browser click <发布日志标签页 ref>
sleep 1
agent-browser click <状态筛选下拉 ref>
sleep 1
agent-browser click <"成功" 选项 ref>
sleep 1
```

**预期：** 表格只显示成功的发布记录

## TC-07-04: 清理日志

```bash
agent-browser click <清理日志按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 弹出清理对话框：
- 日志类型选择（发布日志/管理员日志）
- 时间范围选择（超 3 个月/半年/1 年）
- 警告提示
- `button "确定"`、`button "取消"`
