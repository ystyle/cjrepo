# 前端功能测试流程

使用 agent-browser 对管理后台进行功能验证。测试文件按编号排序，推荐按顺序执行。

## 环境准备

```bash
# 启动服务器（需先构建）
export CJREPO_ADMIN_KEY=your-secret-key
./cjrepo

# 测试时使用的管理密钥
export ADMIN_KEY=your-secret-key
```

## 基本命令速记

```bash
# 导航
agent-browser open http://localhost:8060/admin/teams

# 获取页面可交互元素
agent-browser snapshot -i

# 点击（通过 snapshot 返回的 ref）
agent-browser click e12

# 输入文本
agent-browser fill e4 "text"

# 键入（不清空已有内容）
agent-browser type e4 "text"

# 查看控制台日志/错误
agent-browser console
agent-browser errors

# 清除控制台
agent-browser console --clear
agent-browser errors --clear

# 截图
agent-browser screenshot
```

## 注意事项

1. **每次测试前**先 `agent-browser close --all` 关闭旧会话
2. **每次操作后**用 `agent-browser snapshot -i` 确认页面状态
3. **ref 编号每次打开都可能变化**，测试时需根据实际 snapshot 结果调整
4. 测试管理后台先登录获取 token，后续所有页面基于该会话
