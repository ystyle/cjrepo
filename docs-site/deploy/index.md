# 部署指南

## 自动构建

推送 tag（如 `v1.1.0`）时，GitHub Actions 会自动构建并发布到 [GitHub Releases](https://github.com/ystyle/cjrepo/releases)。

## 二进制部署

### Linux / macOS

从 [GitHub Releases](https://github.com/ystyle/cjrepo/releases) 下载对应平台的二进制文件：

```bash
# 下载二进制
wget https://github.com/ystyle/cjrepo/releases/download/v1.1.0/cjrepo-linux-amd64 -O cjrepo
chmod +x cjrepo

# 运行
export CJREPO_ADMIN_KEY=your-secret-key
./cjrepo
```

### 生产建议

- 使用 systemd 或 supervisor 管理进程
- 定期备份 `./data/cjrepo.db`（SQLite 数据库）
- 存储目录 `./storage/` 存放所有包文件，也需备份
- 反向代理建议：Nginx 添加 HTTPS 支持

## Docker 部署

详见 [Docker 部署](/deploy/docker)。

## 环境变量

详见 [环境变量](/deploy/env)。
