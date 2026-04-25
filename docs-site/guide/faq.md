# 常见问题

## 管理员相关

### 如何登录管理后台？

访问 `http://localhost:8060/admin`，使用环境变量 `CJREPO_ADMIN_KEY` 设置的值登录。

### 忘记管理员密钥怎么办？

查看 `docker-compose.yml` 中的 `CJREPO_ADMIN_KEY`，修改后重启服务：

```bash
docker-compose restart cjrepo
```

## 发布下载

### 发布失败：401 Unauthorized

Token 无效或未配置。使用以下命令查看用户 Token：

```bash
docker-compose exec cjrepo ./cjrepo user list
```

或手动安装时：

```bash
./cjrepo user list
```

更新 `cangjie-repo.toml` 中的 `token` 字段。

### 下载失败：404 Not Found

包不存在或版本不匹配。更新索引后重试：

```bash
cjpm update <pkgname>
```

### 强制认证模式

默认仅发布需要认证。设置 `CJREPO_REQUIRE_AUTH=true` 后，下载和索引也需要 Token：

```yaml
environment:
  - CJREPO_ADMIN_KEY=your-admin-key
  - CJREPO_REQUIRE_AUTH=true
```

适用于私有仓库场景，防止未授权访问。

## Docker

### 容器无法启动

```bash
# 查看日志
docker-compose logs cjrepo

# 检查端口占用
lsof -i :8060

# 重新构建
docker-compose down
docker-compose up -d --build
```

## SDK

### 需要什么版本的仓颉 SDK？

**v1.1.0** 或更高版本（[下载地址](https://cangjie-lang.cn/download)）。

低于此版本的 SDK 可能不支持 `cjpm publish`、`cjpm install`、`cjpm update` 等功能。

## 依赖缓存

`~/.cjpm/repository/` 下存放下载的包和索引：

- `index/` — 索引文件
- `source/` — 源码包

如需重新下载，删除对应目录重新执行 `cjpm check`：

```bash
rm -rf ~/.cjpm/repository/source/{org}/{name}-{version}
cjpm check
```
