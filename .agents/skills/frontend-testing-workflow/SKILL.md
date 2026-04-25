---
name: frontend-testing-workflow
description: 基于 agent-browser 的前端功能测试流程定义。用于需要编写前端功能测试场景、定义测试步骤和预期结果、或使用 agent-browser 自动化执行前端验证的场景。适用任意 Web 前端项目。
---

# Frontend Testing Workflow

基于 agent-browser 的前端功能测试框架。定义可复用的测试流程规范，包含测试场景组织方式、用例格式、执行流程。

## 何时使用

- 需要为前端功能模块编写测试场景
- 需要定义 agent-browser 自动化测试步骤
- 需要建立项目级前端测试规范
- 需要验收前端功能是否正常工作

## 目录结构

```
tests/web/                   # 前端测试流程目录
├── README.md                # 环境准备、命令速记
├── 00-auth-login.md         # 登录模块测试
├── 01-feature-a.md          # 功能模块 A
├── 02-feature-b.md          # 功能模块 B
└── ...
```

### 命名规范

| 文件 | 规则 |
|------|------|
| 目录 | `tests/web/` |
| 文件 | `{编号}-{模块名}.md`，编号两位数字（00-99） |
| 测试用例 | `TC-{文件编号}-{序号}: 描述` |

## 用例格式

每个 `.md` 文件的结构：

```markdown
# 模块名称

简要描述模块功能。

## 前置条件

- 条件 1
- 条件 2

## TC-XX-01: 页面加载验证

```bash
agent-browser open http://localhost:8060/admin/<path>
sleep 2
agent-browser snapshot -i
```

**预期：**
- 页面标题应为"..."
- 应包含 xxxx 元素
- `agent-browser console` 无报错
- `agent-browser errors` 无报错

## TC-XX-02: 操作功能

```bash
agent-browser console --clear
agent-browser click <操作按钮 ref>
sleep 2
agent-browser snapshot -i
```

**预期：**
- 应出现 xxx 对话框
- 应包含 xxx 表单字段
- `agent-browser console` 无报错
```

## 核心流程

### 阶段 1：页面加载验证

每次导航到新页面时，先确认：
1. 页面标题正确
2. 关键元素存在
3. 控制台无报错

```bash
# 打开页面
agent-browser open http://localhost:8060/admin/<path>

# 等待渲染
sleep 2

# 获取页面结构
agent-browser snapshot -i

# 检查控制台
agent-browser console
agent-browser errors
```

### 阶段 2：操作执行

单个操作步骤的规范：

```bash
# 1. 清除之前的日志
agent-browser console --clear
agent-browser errors --clear

# 2. 执行操作（点击/输入等）
agent-browser click <ref>
agent-browser fill <ref> "value"

# 3. 等待响应
sleep 2

# 4. 验证结果
agent-browser snapshot -i
agent-browser console
```

### 阶段 3：结果验证

验证点类型：

| 验证内容 | 方法 |
|----------|------|
| 页面元素 | `agent-browser snapshot -i` 查看是否存在目标元素 |
| 控制台报错 | `agent-browser console` 应为空或无 error |
| 页面报错 | `agent-browser errors` 应为空 |
| 对话框内容 | `snapshot -i` 检查 dialog/input/button 等 |
| 表格数据 | 检查 cell 内容是否匹配 |
| URL 跳转 | 观察 open 命令返回的 URL |

## Ref 处理

agent-browser 每次打开页面时 ref 编号都会变化，因此：

1. **测试文档中用描述性占位符**：
   `<创建用户按钮 ref>`、`<搜索框 ref>`、`<确定按钮 ref>`

2. **执行时先用 snapshot 获取实际 ref**：
   ```bash
   agent-browser snapshot -i
   # 输出: - button "创建用户" [ref=e12]
   # 然后: agent-browser click e12
   ```

3. **常见的 ref 特征**：
   - `button "xxx"` → 按钮
   - `textbox "xxx"` → 文本输入框
   - `cell "xxx"` → 表格单元格
   - `menuitem "xxx"` → 菜单项

## 常见 agent-browser 命令

```bash
agent-browser open <url>           # 导航
agent-browser snapshot -i          # 取可交互元素列表
agent-browser click <ref>          # 点击
agent-browser fill <ref> <text>    # 清空并输入
agent-browser type <ref> <text>    # 追加输入（不清空）
agent-browser press <key>          # 按键（Enter, Tab, Escape 等）
agent-browser console              # 查看控制台日志
agent-browser console --clear      # 清除控制台日志
agent-browser errors               # 查看页面错误
agent-browser errors --clear       # 清除页面错误
agent-browser screenshot           # 截图
agent-browser wait <ms>            # 等待
agent-browser close --all          # 关闭所有会话
```

## 常用检查模式

### 检查对话框

```bash
snapshot -i | grep -E "dialog|textbox|button.*确定|button.*取消"
```

期望包含：对话框标题（可通过 grep title）、输入框、确定/取消按钮

### 检查表格数据

```bash
snapshot -i | grep "cell"
```

期望包含：对应数据单元格

### 检查页面是否干净

```bash
errors     # 应返回空
console    # 应无 error 级别日志，或有预期内的 warning
```

## 完整示例

以下是一个完整的登录测试流程示例：

### TC-00-01: 未登录重定向

```bash
agent-browser close --all
agent-browser open http://localhost:8060/admin/teams
```

**预期：** URL 跳转到 `http://localhost:8060/admin/login?redirect=/admin/teams`

### TC-00-02: 登录成功

```bash
agent-browser snapshot -i                    # 获取登录页元素
agent-browser fill <密钥输入框 ref> "$ADMIN_KEY"   # 输入密钥
agent-browser click <登录按钮 ref>
sleep 3
agent-browser snapshot -i                    # 验证进入后台
```

**预期：** 菜单项可见，console 无报错

## 与其他技能配合

- **agent-browser skill（core）**：进阶用法、技巧、故障排查
- **测试数据准备**：可通过 `curl` 或 `hurl` 预处理数据，再用 agent-browser 验证前端展示
