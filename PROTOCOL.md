# CJRepo 技术文档

本文档描述 CJRepo 的协议规范、架构设计和实现细节。

## 目录

- [协议规范](#协议规范)
- [系统架构](#系统架构)
- [数据模型](#数据模型)
- [API 端点](#api-端点)
- [实现细节](#实现细节)
- [部署指南](#部署指南)

## 协议规范

### 发布协议

CJRepo 完全实现了 `cjpm publish` 的自定义二进制协议，与 `cjpm publish` 完全兼容。

#### 数据格式

```
[meta-data 部分]
├── 版本号 (1 byte): 0x01
├── 大小 (4 bytes): 小端序 int32
└── 数据: meta-data.json 内容

[tarball 部分]
├── 版本号 (1 byte): 0x01
├── 大小 (4 bytes): 小端序 int32
└── 数据: .cjp 文件内容
```

#### meta-data.json 结构

```json
{
  "organization": "default",
  "name": "mypackage",
  "version": "1.0.0",
  "description": "My package",
  "artifact-type": "src",
  "executable": false,
  "authors": [],
  "repository": "",
  "homepage": "",
  "documentation": "",
  "tag": [],
  "category": [],
  "license": [],
  "index": {
    "sha256sum": "abc123...",
    "dependencies": [],
    "test-dependencies": [],
    "script-dependencies": [],
    "yanked": false,
    "cjc-version": "0.53.1",
    "index-version": 1
  },
  "meta-version": 1
}
```

### 文件格式

#### .cjp 文件格式

- **格式**：tar.gz
- **结构**：`{name}-{version}/` 目录包含源码
- **内容**：
  - `cjpm.toml` - 包配置
  - `src/` - 源代码目录
  - `README.md` - 说明文档

示例：
```
defer-1.0.0/
├── cjpm.toml
├── src/
│   └── main.cj
└── README.md
```

#### tar.gz 打包参数

```cj
TarGzip.archive(
    fromDir: targetCopyDir,
    destFile: tarPath,
    includeBaseDirectory: true  // 包含基础目录
)
```

## 系统架构

### 整体架构

```
┌─────────────┐
│  cjpm 客户端  │
└──────┬──────┘
       │ HTTP/1.1
       ↓
┌─────────────┐
│  Fiber HTTP  │
│   服务器    │
└──────┬──────┘
       │
       ├─────────────────┬────────────────┐
       ↓                 ↓                ↓
┌──────────┐    ┌──────────┐   ┌──────────────┐
│Protocol  │    │ Handlers │   │   SQLite    │
│ Parser   │    │          │   │    Database  │
└──────────┘    └──────────┘   └──────────────┘
                      │
                      ↓
              ┌──────────────┐
              │   文件系统    │
              │  storage/    │
              └──────────────┘
```

### 核心组件

#### 1. Protocol Parser (protocol/parser.go)

**职责**：解析二进制上传数据

```go
func ParsePublishData(data []byte) (*PublishRequest, error)
```

**处理流程**：
1. 读取 meta-data 部分（版本号 + 大小 + 数据）
2. 读取 tarball 部分（版本号 + 大小 + 数据）
3. 验证格式和大小限制（最大 500MB）
4. 返回解析结果

#### 2. Handlers (handlers/)

**PublishHandler** (handlers/publish.go)
- 验证 Token
- 解析二进制数据
- 验证 SHA256 校验和
- 检查重复发布
- 保存文件和数据库记录

**DownloadHandler** (handlers/download.go)
- 查询数据库获取包信息
- 读取文件内容
- 返回 tar.gz 文件（`application/x-gzip`）
- 设置 `Connection: close` 避免持久连接问题

**IndexHandler** (handlers/index.go)
- 解析路径参数（`/index/{first}/{second}/{name}`）
- 查询数据库获取包版本
- 返回 ArtifactIndex 格式的 JSON Lines

#### 3. Storage Manager (storage/manager.go)

**职责**：管理包文件的存储路径

- **路径格式**：`storage/{org}/{name}/{version}.cjp`
- **绝对路径**：使用绝对路径避免 Fiber 找不到文件
- **自动创建**：自动创建不存在的目录

## 数据模型

### Package 表

```go
type Package struct {
    ID            int64     `xorm:"pk autoincr"`
    Organization  string    `xorm:"index"`
    Name          string    `xorm:"index"`
    Version       string    `xorm:"index"`
    Description   string
    ArtifactType  string
    Executable    bool
    Authors       string    // JSON array
    Repository    string
    Homepage      string
    Documentation string
    Tags          string    // JSON array
    Categories    string    // JSON array
    Licenses      string    // JSON array
    MetaVersion   int
    MetaData      string    `xorm:"text"`      // 完整 meta-data.json
    TarballPath   string    `xorm:"text"`
    TarballSize   int64
    TarballSHA256 string    `xorm:"index"`
    CreatedAt     time.Time `xorm:"created"`
    UpdatedAt     time.Time `xorm:"updated"`
}
```

### User 表

```go
type User struct {
    ID        int64     `xorm:"pk autoincr"`
    Username  string    `xorm:"unique"`
    Token     string    `xorm:"unique index"`
    Email     string
    IsActive  bool      `xorm:"default true"`
    CreatedAt time.Time `xorm:"created"`
}
```

### PublishLog 表

```go
type PublishLog struct {
    ID           int64     `xorm:"pk autoincr"`
    Organization string
    PackageName  string
    Version      string
    Status       string    // success/failed
    ErrorMessage string    `xorm:"text"`
    IPAddr       string
    UserAgent    string
    CreatedAt    time.Time `xorm:"created"`
}
```

## API 端点

### 1. 发布包

**端点**：`POST /pkg/{name}?organization={org}`

**请求头**：
```
Authorization: {token}
Content-Type: application/octet-stream
```

**请求体**：自定义二进制格式

**响应**：
```json
{
  "message": "package published successfully",
  "package": {
    "organization": "default",
    "name": "mypackage",
    "version": "1.0.0"
  }
}
```

**状态码**：
- 200 - 成功
- 400 - 参数错误
- 401 - 未认证
- 403 - 权限不足
- 404 - 包不存在（用于下载）
- 409 - 版本已存在
- 500 - 服务器错误

### 2. 下载包

**端点**：`GET /pkg/{name}/{version}?organization={org}`

**响应头**：
```
Content-Type: application/x-gzip
Content-Length: {size}
Content-Disposition: attachment; filename="{name}-{version}.cjp"
Connection: close
```

**响应体**：.cjp 文件内容（tar.gz 格式）

### 3. 索引查询

**端点**：`GET /index/{first}/{second}/{name}?organization={org}`

**路径参数**：
- `first`: 包名前 2 个字符（0:2）
- `second`: 包名第 2-4 个字符（2:4）
- `name`: 完整包名

**示例**：包名 "defer" → `/index/de/fe/defer`

**响应**：`Content-Type: application/x-ndjson`

每行一个 ArtifactIndex 对象：
```json
{"organization":"default","name":"defer","version":"1.0.0","dependencies":[],"testDependencies":[],"scriptDependencies":[],"sha256sum":"abc...","yanked":false,"cjc-version":"0.59.6","index-version":1}
```

### 4. 健康检查

**端点**：`GET /health`

**响应**：
```json
{"status": "ok"}
```

## 实现细节

### 二进制协议解析

**关键代码** (protocol/parser.go:29-68)：

```go
func ParsePublishData(data []byte) (*PublishRequest, error) {
    reader := bytes.NewReader(data)
    req := &PublishRequest{}

    // 解析 meta-data 部分
    metaData, err := readSection(reader, META_DATA_VERSION, "meta-data")
    if err != nil {
        return nil, fmt.Errorf("failed to parse meta-data: %w", err)
    }
    req.MetaData = metaData

    // 解析 tarball 部分
    tarball, err := readSection(reader, TARBALL_VERSION, "tarball")
    if err != nil {
        return nil, fmt.Errorf("failed to parse tarball: %w", err)
    }
    req.Tarball = tarball

    return req, nil
}

func readSection(reader *bytes.Reader, expectedVersion byte, sectionType string) ([]byte, error) {
    // 读取版本号 (1 byte)
    version, err := reader.ReadByte()
    if version != expectedVersion {
        return nil, fmt.Errorf("invalid version: expected %d, got %d", expectedVersion, version)
    }

    // 读取大小 (4 bytes, 小端序)
    var size int32
    binary.Read(reader, binary.LittleEndian, &size)

    // 读取数据
    data := make([]byte, size)
    reader.Read(data)

    return data, nil
}
```

### SHA256 校验

**关键代码** (handlers/publish.go:133-141)：

```go
actualSHA256 := protocol.CalculateSHA256(req.Tarball)
log.Printf("[DEBUG] SHA256 - expected: %s, actual: %s", expectedSHA256, actualSHA256)

if !protocol.ValidateTarballSHA256(req.Tarball, expectedSHA256) {
    h.logPublish(organization, packageName, metaData.Version, "failed", "checksum mismatch", c)
    return c.Status(400).JSON(fiber.Map{
        "error": "tarball checksum mismatch",
    })
}
```

### 文件存储

**路径管理** (storage/manager.go:14-24)：

```go
func NewStorageManager(rootPath string) *Manager {
    // 转换为绝对路径
    absPath, err := filepath.Abs(rootPath)
    if err != nil {
        absPath = rootPath
    }
    return &Manager{
        rootPath: absPath,
    }
}
```

**文件保存** (storage/manager.go:38-50)：

```go
func (m *Manager) SaveTarball(org, name, version string, data []byte) error {
    // 确保目录存在
    if err := m.EnsurePath(org, name); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // 保存文件
    path := m.GetTarballPath(org, name, version)
    if err := os.WriteFile(path, data, 0644); err != nil {
        return fmt.Errorf("failed to write file: %w", err)
    }

    return nil
}
```

### 下载响应处理

**关键点**：
1. **使用 `c.Send(data)` 而非 `c.SendFile()`**：确保数据完整传输
2. **设置 `Connection: close`**：避免持久连接导致的读取问题
3. **直接读取文件到内存**：避免流式传输的边界问题

```go
// 读取整个文件到内存
data, err := os.ReadFile(pkg.TarballPath)

// 设置响应头
c.Set("Content-Type", "application/x-gzip")
c.Set("Content-Disposition", "attachment; filename=\""+packageName+"-"+version+".cjp\"")
c.Set("Content-Length", fmt.Sprintf("%d", len(data)))
c.Set("Connection", "close")

// 直接发送数据
return c.Send(data)
```

### Index 响应格式

**关键点**：返回 `ArtifactIndex` 结构，而非嵌套的 meta-data

```go
// 构建 ArtifactIndex 结构
artifactIndex := fiber.Map{
    "organization":      pkg.Organization,
    "name":              pkg.Name,
    "version":           pkg.Version,
    "dependencies":      []interface{}{},
    "testDependencies":  []interface{}{},
    "scriptDependencies": []interface{}{},
    "sha256sum":         indexField.(map[string]interface{})["sha256sum"],
    "yanked":           false,
    "cjc-version":       indexField.(map[string]interface{})["cjc-version"],
    "index-version":     1,
}
```

## 关键问题与解决方案

### 问题 1: xorm 列名映射错误

**现象**：`no such column: organization`

**原因**：xorm 标签写错为 `xorm:"index 'org'"`，被解释为列名

**解决**：
```go
// 错误
Organization string `xorm:"index 'org'"`

// 正确
Organization string `xorm:"index"`
```

### 问题 2: JSON 标签不匹配

**现象**：无法解析 meta-data.json

**原因**：仓颉使用 kebab-case（`artifact-type`），Go 默认 camelCase

**解决**：
```go
type MetaData struct {
    ArtifactType string `json:"artifact-type"`  // 注意 kebab-case
    MetaVersion  int    `json:"meta-version"`
}
```

### 问题 3: Index 返回格式错误

**现象**：SHA256 校验失败

**原因**：返回嵌套的 meta-data，而 cjpm 期望扁平的 ArtifactIndex

**解决**：直接构建 ArtifactIndex 结构，不使用嵌套的 `index` 字段

### 问题 4: 相对路径导致文件找不到

**现象**：下载时找不到文件

**原因**：存储使用相对路径 `storage/...`

**解决**：使用 `filepath.Abs()` 转换为绝对路径

### 问题 5: Socket 关闭导致下载失败

**现象**：`ConnectionException: Socket is closed`

**原因**：
1. Fiber 的 `c.SendFile()` 可能有兼容性问题
2. `Connection: keep-alive` 导致的流式读取问题

**解决**：
1. 使用 `c.Send(data)` 直接发送完整数据
2. 设置 `Connection: close` 强制关闭连接
3. 设置正确的 `Content-Length`

## 性能优化

1. **数据库连接池**：
   ```go
   engine.SetMaxOpenConns(10)
   engine.SetMaxIdleConns(5)
   ```

2. **日志级别**：
   ```go
   app.Use(logger.New(logger.Config{
       Format: "[${time}] ${status} - ${method} ${path}\n",
   }))
   ```

3. **文件缓存**：考虑使用 CDN 或对象存储服务（如 S3）

## 安全考虑

### 1. Token 认证

- 所有写操作（publish）必须验证 Token
- Token 存储在数据库中，使用 `SELECT ... WHERE token = ? AND is_active = 1` 验证
- 建议生产环境使用更安全的 Token 生成方式（如 UUID + HMAC）

### 2. 文件大小限制

```go
const MAX_FILE_SIZE = 500 * 1024 * 1024 // 500MB
```

### 3. 路径安全

- 使用 `filepath.Abs()` 避免路径遍历
- 不信任用户输入的文件名

### 4. 输入验证

- 验证包名格式
- 验证版本号格式
- 验证 organization

### 5. 审计日志

- 记录所有发布操作到 `publish_logs` 表
- 包含 IP、User-Agent、时间戳等信息

## 扩展性

### 支持新的存储后端

实现 `Storage` 接口：
```go
type Storage interface {
    SaveTarball(org, name, version string, data []byte) error
    GetTarballPath(org, name, version string) string
    DeleteTarball(org, name, version string) error
}
```

### 支持对象存储

集成 S3、OSS 等对象存储服务：
```go
type S3Storage struct {
    bucket string
    region string
}

func (s *S3Storage) SaveTarball(org, name, version string, data []byte) error {
    key := fmt.Sprintf("%s/%s/%s.cjp", org, name, version)
    // 上传到 S3
}
```

### 支持 Webhooks

发布成功后触发 Webhook：
```go
func notifyWebhooks(pkg *models.Package) error {
    // 调用配置的 Webhook URL
    for _, url := range webhookUrls {
        http.Post(url, "application/json", pkg)
    }
    return nil
}
```

## 故障排查

### 日志查看

```bash
# 查看服务日志
journalctl -u cjrepo -f

# 查看数据库
sqlite3 data/cjrepo.db "SELECT * FROM publish_logs ORDER BY created_at DESC LIMIT 20;"
```

### 性能分析

```bash
# CPU 性能分析
go tool pprof http://localhost:8060/debug/pprof/profile

# 内存分析
go tool pprof http://localhost:8060/debug/pprof/heap
```

### 监控指标

建议添加以下监控：
- 请求响应时间
- 发布成功率
- 下载成功率
- 数据库连接数
- 磁盘使用率

## 相关资源

- [Fiber 文档](https://docs.gofiber.io/)
- [xorm 文档](https://xorm.io/)
- [仓颉编程语言](https://docs.cangjie-lang.cn/)
- [cjpm 源码](https://gitlink.com.cn/cangjie/cjpm)
