# Docker 部署

## Docker Compose（推荐）

```yaml
version: '3.8'
services:
  cjrepo:
    image: ghcr.io/ystyle/cjrepo:latest
    ports:
      - "8060:8060"
    volumes:
      - ./data:/app/data
      - ./storage:/app/storage
    environment:
      - CJREPO_ADMIN_KEY=your-secret-key
      - TZ=Asia/Shanghai
    restart: unless-stopped
```

## 构建镜像

```bash
git clone https://github.com/ystyle/cjrepo.git
cd cjrepo
docker build -t cjrepo .
docker run -d \
  -p 8060:8060 \
  -v ./data:/app/data \
  -v ./storage:/app/storage \
  -e CJREPO_ADMIN_KEY=your-secret-key \
  cjrepo
```

## 数据持久化

| 目录 | 说明 | 必需备份 |
|------|------|---------|
| `/app/data` | SQLite 数据库 | ✅ |
| `/app/storage` | 包文件存储 | ✅ |
