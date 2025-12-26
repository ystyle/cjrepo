# CJRepo - 仓颉中央库服务

一个兼容 `cjpm` 的仓颉包中央库服务，支持包的发布、下载和索引查询。

## 快速开始

### 1. 安装

```bash
# 克隆项目
git clone <repository-url>
cd cjrepo

# 构建服务
go build -o cjrepo main.go
```

### 2. 启动服务

```bash
# 启动服务器（默认端口 8060）
./cjrepo

# 服务将在 http://localhost:8060 启动
```

### 3. 添加用户

```bash
# 创建新用户
./cjrepo user add alice alice@example.com

# 输出示例：
# ✓ User created successfully!
# Token: token-alice-12345
#
# Add this to your cangjie-repo.toml:
# [repository.home]
# registry = "http://localhost:8060"
# token = "token-alice-12345"
```

### 4. 配置 cjpm

创建或编辑项目的 `cangjie-repo.toml`：

```toml
[repository.home]
registry = "http://localhost:8060"
token = "token-alice-12345"
```

或配置全局设置（`$HOME/.cjpm/cangjie-repo.toml` 或 `$CANGJIE_HOME/tools/config/cangjie-repo.toml`）。

## 部署指南

### 开发环境

```bash
# 构建
go build -o cjrepo main.go

# 启动
./cjrepo
```

### 生产环境

#### 使用 systemd

创建 `/etc/systemd/system/cjrepo.service`：

```ini
[Unit]
Description=Cangjie Package Repository
After=network.target

[Service]
Type=simple
User=cjrepo
WorkingDirectory=/opt/cjrepo
ExecStart=/opt/cjrepo/cjrepo
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable cjrepo
sudo systemctl start cjrepo
```

#### 使用 Docker（推荐）

**使用 Docker Compose**：
```bash
# 构建并启动
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

**使用 Docker 命令**：
```bash
# 构建镜像
docker build -t cjrepo:latest .

# 运行容器
docker run -d \
  --name cjrepo \
  -p 8060:8060 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/storage:/app/storage \
  --restart unless-stopped \
  cjrepo:latest

# 在容器中执行用户管理
docker exec -it cjrepo ./cjrepo user add alice alice@example.com
```

**Dockerfile 特点**：
- 第一阶段：使用 golang:1.23-alpine 编译
- 第二阶段：使用 alpine:latest 运行，镜像体积小
- 非 root 用户运行（cjrepo:1000）
- 包含健康检查
- 数据持久化到宿主机 volume

#### 使用 Nginx 反向代理

创建 `/etc/nginx/sites-available/cjrepo`：

```nginx
server {
    listen 80;
    server_name repo.example.com;

    location / {
        proxy_pass http://localhost:8060;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 上传大小限制
        client_max_body_size 500M;
    }
}
```

### 环境变量

支持通过环境变量配置：

```bash
export CJREPO_DB_PATH=/data/cjrepo.db
export CJREPO_STORAGE_PATH=/storage
export CJREPO_PORT=8060

./cjrepo
```

### 容器内的用户管理

```bash
# 在运行的容器中执行命令
docker-compose exec cjrepo ./cjrepo user add username email@example.com
docker-compose exec cjrepo ./cjrepo user list
docker-compose exec cjrepo ./cjrepo user delete username
```

### 配置 cjpm 连接 Docker 服务

如果 cjpm 在宿主机运行，而 cjrepo 在容器中运行，配置如下：

```toml
[repository.home]
registry = "http://localhost:8060"
token = "token-alice-12345"
```

如果 cjpm 和 cjrepo 都在 Docker 网络中，需要使用容器名称：

```toml
[repository.home]
registry = "http://cjrepo:8060"
token = "token-alice-12345"
```

## 仓颉项目使用指南

### 创建新项目

```bash
# 初始化项目
cjpm init --name myproject
cd myproject
```

### 添加依赖

编辑 `cjpm.toml`：

```toml
[dependencies]
defer = { version = "1.0.0" }
```

### 下载依赖

```bash
# 检查并自动下载依赖
cjpm check

# 输出：
# The valid serial compilation order is:
#   defer
# cjpm check success
```

### 编译项目

```bash
# 编译
cjpm build

# 编译并运行
cjpm run
```

### 发布包

```bash
# 1. 打包（生成 .cjp 文件和 meta-data.json）
cjpm bundle

# 2. 发布到中央库
cjpm publish

# 成功输出：
# mypackage-1.0.0 publish success
```

### 更新索引

```bash
# 更新所有依赖的索引
cjpm update

# 更新特定包的索引
cjpm update defer
```

## 命令参考

### 服务命令

```bash
./cjrepo                    # 启动服务器
./cjrepo help              # 显示帮助
./cjrepo version           # 显示版本
```

### 用户管理

```bash
./cjrepo user add <username> <email>     # 创建用户
./cjrepo user list                       # 列出所有用户
./cjrepo user delete <username>          # 删除用户
```

### cjpm 命令

```bash
cjpm init --name <name>        # 初始化项目
cjpm check                     # 检查并下载依赖
cjpm update                    # 更新索引
cjpm bundle                    # 打包
cjpm publish                   # 发布包
cjpm build                     # 编译
cjpm run                       # 运行
```

## 常见问题

### 1. 发布失败：401 Unauthorized

**原因**：Token 无效或未配置

**解决**：
```bash
# 检查 token
./cjrepo user list

# 更新 cangjie-repo.toml 中的 token
```

### 2. 下载失败：404 Not Found

**原因**：包不存在或版本不匹配

**解决**：
```bash
# 更新索引
cjpm update <pkgname>

# 检查可用版本
```

### 3. 端口被占用

```bash
# 查找占用进程
lsof -i :8060

# 杀死进程
killall cjrepo

# 重新启动
./cjrepo
```

### 4. 依赖缓存位置

依赖下载在 `~/.cjpm/repository/`：
- `index/` - 索引文件
- `source/` - 源码包

如需重新下载，删除对应的目录即可：
```bash
rm -rf ~/.cjpm/repository/source/{org}/{name}-{version}
cjpm check
```

## 数据库管理

```bash
# 查看已发布的包
sqlite3 data/cjrepo.db "SELECT organization, name, version, description FROM packages;"

# 查看用户
sqlite3 data/cjrepo.db "SELECT username, email, token FROM users;"

# 查看发布日志
sqlite3 data/cjrepo.db "SELECT * FROM publish_logs ORDER BY created_at DESC LIMIT 10;"
```

## 项目结构

```
cjrepo/
├── main.go              # 主入口（包含用户管理）
├── internal/            # 内部实现
│   ├── handlers/        # HTTP 处理器
│   ├── models/          # 数据模型
│   ├── protocol/        # 协议解析
│   └── storage/         # 文件存储管理
├── data/                # 数据库文件
├── storage/             # 包文件存储
├── Dockerfile           # Docker 镜像构建
├── docker-compose.yml   # Docker Compose 配置
└── .dockerignore        # Docker 忽略文件
```

## 技术文档

详细的协议规范、架构设计和实现细节请参阅：

- **[PROTOCOL.md](PROTOCOL.md)** - 协议规范与架构设计
- **[README.md](README.md)** - 本文档

## 许可证

MIT License

## 相关资源

- [仓颉编程语言官方文档](https://docs.cangjie-lang.cn/)
- [cjpm 使用指南](https://docs.cangjie-lang.cn/docs/latest/tools/source_zh_cn/tools/cjpm_manual_cjnative_community.html)
