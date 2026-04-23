# 第一阶段：构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

# 设置阿里云 Alpine 镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 设置阿里云 npm 镜像源
RUN npm config set registry https://registry.npmmirror.com

# 复制前端项目文件
COPY frontend/package.json frontend/pnpm-lock.yaml ./

# 安装 pnpm 和依赖
RUN npm install -g pnpm --registry=https://registry.npmmirror.com
RUN pnpm install --frozen-lockfile

# 复制源代码
COPY frontend/ ./

# 构建前端
RUN pnpm build-only

# 第二阶段：构建 Go 应用
FROM golang:1.23-alpine AS go-builder

WORKDIR /build

# 设置 Go 中国代理
ENV GOPROXY=https://goproxy.cn,direct

# 设置阿里云 Alpine 镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# 复制 go.mod 和 go.sum
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download || true

# 复制源代码和前端构建产物
COPY . .
COPY --from=frontend-builder /frontend/dist ./frontend/dist

# 构建应用（包含嵌入的前端文件和构建信息）
RUN BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S') && \
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    GIT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "dev") && \
    echo "Build info: $BUILD_DATE, $GIT_COMMIT, $GIT_VERSION" && \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-w -s -X 'ystyle.top/go/cjrepo/internal/version.buildDate=$BUILD_DATE' -X 'ystyle.top/go/cjrepo/internal/version.gitCommit=$GIT_COMMIT' -X 'ystyle.top/go/cjrepo/internal/version.gitVersion=$GIT_VERSION'" \
    -o cjrepo .

# 第三阶段：运行阶段
FROM alpine:latest

# 设置阿里云 Alpine 镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装运行时依赖
RUN apk add --no-cache ca-certificates sqlite-libs

# 创建非 root 用户
RUN addgroup -g 1000 cjrepo && \
    adduser -D -u 1000 -G cjrepo cjrepo

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=go-builder /build/cjrepo .

# 创建必要的目录
RUN mkdir -p data storage && \
    chown -R cjrepo:cjrepo /app

# 切换到非 root 用户
USER cjrepo

# 暴露端口
EXPOSE 8060

# 设置默认命令
CMD ["./cjrepo"]
