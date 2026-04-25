# 仪表盘

验证仪表盘页面功能。

## 前置条件

- 已登录进入管理后台

## TC-01-01: 仪表盘正确加载

登录后默认跳转到仪表盘，或直接导航：

```bash
agent-browser open http://localhost:8060/admin/dashboard
sleep 2
agent-browser snapshot -i
```

**预期：**
- 页面标题为"仪表盘"
- 包含统计卡片（包总数、版本总数、用户总数等）
- 包含快捷入口卡片（包管理、用户管理、操作日志）
- `agent-browser console` 无报错
- `agent-browser errors` 无报错

## TC-01-02: 快捷入口导航

```bash
# 点击"包管理"快捷入口
agent-browser click <包管理卡片 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 导航到 `/admin/packages`，页面标题为"包管理"

```bash
# 点击菜单返回仪表盘
agent-browser click <仪表盘菜单 ref>
sleep 2
agent-browser snapshot -i
```

**预期：** 回到仪表盘页面
