# CLAUDE.md

写一个最小化的仓颉中央库的服务

## 仓颉cjpm上传代码
- 命令入口: `/home/ystyle/Code/CangJie/office/cangjie_tools/cjpm/src/command/publish.cj`
- 命令实现：`/home/ystyle/Code/CangJie/office/cangjie_tools/cjpm/src/implement/publish.cj`
- 命令上传: `/home/ystyle/Code/CangJie/office/cangjie_tools/cjpm/src/implement/depot.cj`

## 构建物
- cjp包: `~/Code/CangJie/defer-cj/target/defer-1.0.0.cjp`
- 元数据： ` ~/Code/CangJie/defer-cj/target/meta-data.json`

## 系统要求
- 使用zsh执行cd

## 仓颉中央库服务

### 项目结构
```
/home/ystyle/Code/Go/cjrepo/
├── models/          # 数据模型定义
│   └── package.go   # Package, User, PublishLog
├── protocol/        # 协议解析
│   └── parser.go    # 二进制协议解析器
├── handlers/        # HTTP处理器
│   ├── publish.go   # 发布端点
│   ├── download.go  # 下载端点
│   └── index.go     # 索引端点
├── storage/         # 文件存储管理
│   └── manager.go   # 存储路径管理
├── data/            # 数据库文件
├── storage/         # 包文件存储
├── main.go          # 主入口
└── add_user.go      # 用户管理工具
```

### 协议说明

#### cjpm publish 二进制协议
```
[meta-data部分]
├── 版本号 (1 byte): 0x01
├── 大小 (4 bytes): 小端序 int32
└── 数据: meta-data.json 内容

[tarball部分]
├── 版本号 (1 byte): 0x01
├── 大小 (4 bytes): 小端序 int32
└── 数据: .cjp 文件内容
```

#### HTTP 端点
1. **POST /pkg/{name}?organization={org}** - 发布包
2. **GET /pkg/{name}/{version}?organization={org}** - 下载包
3. **GET /index/{first}/{second}/{name}?organization={org}** - 获取索引
4. **GET /health** - 健康检查

### 使用方法

#### 1. 构建服务
```bash
go build -o cjrepo main.go
```

#### 2. 启动服务
```bash
./cjrepo
# 服务将在 http://localhost:8060 启动
```

#### 3. 添加测试用户
```bash
go run add_user.go testuser test@example.com
# 将输出: Authorization: token-testuser-xxxxx
```

#### 4. 配置 cjpm
编辑 `~/.cjr/config.toml` 或项目的 `cangjie-repo.toml`:
```toml
[repository.home]
registry = "http://localhost:8060"
token = "token-testuser-xxxxx"
```

#### 5. 发布包
```bash
cjpm publish
```

### 依赖
- github.com/gofiber/fiber/v2
- xorm.io/xorm
- github.com/mattn/go-sqlite3
- github.com/google/uuid