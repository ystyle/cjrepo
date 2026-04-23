# AGENTS.md

## 项目概述

仓颉语言包私有仓库服务（cjrepo），兼容 cjpm 的发布/下载/索引协议。 协议在：`docs/中心仓通信规格.md`

## 相关项目
- `cjpm 源码`： /home/ystyle/Projects/cangjie_tools/cjpm

## 相关工具
- `gh` 访问github, 回复和提交需要经过用户同意
- 代理： `export https_proxy=http://192.168.3.6:1080`

## 构建

```bash
# 后端
go build -o cjrepo main.go

# 前端（需要先构建前端才能嵌入）
cd frontend && pnpm install && pnpm build && cd ..
```

## 运行

```bash
# 必需环境变量
export CJREPO_ADMIN_KEY=your-secret-key
./cjrepo                  # 启动服务器（默认端口 :8060）

# 用户管理命令
./cjrepo user add <username> <email>
./cjrepo user list
./cjrepo user delete <username>
```

## 架构

```
internal/
├── handlers/      # HTTP 处理器（publish/download/index/admin/public）
├── protocol/      # cjpm 二进制协议解析器
├── storage/       # 文件存储（storage/{org}/{name}/{version}.cjp）
├── models/        # XORM 数据模型
├── auth/          # JWT 认证服务
├── middleware/    # JWT 中间件
└── upstream/      # 上游源代理同步
```

## cjpm 协议端点

- `POST /pkg/{name}?organization={org}` - 发布包（需 Token）
- `GET /pkg/{name}/{version}?organization={org}` - 下载 .cjp 文件
- `GET /index/{first}/{second}/{name}?organization={org}` - 索引查询

## 关键约定

1. **二进制协议格式**：`[version(1byte)][size(4byte le)][data]` 两段（meta-data + tarball）
2. **JSON 字段命名**：meta-data.json 使用 kebab-case（如 `artifact-type`），不是 camelCase
3. **存储路径**：使用绝对路径，格式 `storage/{org}/{name}/{version}.cjp`
4. **下载响应**：必须设置 `Connection: close` 避免 cjpm socket 问题

## 数据库

SQLite (`./data/cjrepo.db`)，表：packages, users, publish_logs, admin_log, upstreams, organizations

## Docker

```bash
docker-compose up -d        # 启动
docker-compose exec cjrepo ./cjrepo user add <name> <email>
```

## 前端开发

```bash
cd frontend
pnpm dev        # 开发服务器
pnpm build      # 构建（嵌入后端）
pnpm format     # 格式化
```

## 测试验证

启动后访问：
- http://localhost:8060/health - 健康检查
- http://localhost:8060/admin - 管理后台（用 CJREPO_ADMIN_KEY 登录）