# 第一阶段：构建前端 + 文档
FROM node:20-alpine AS builder

WORKDIR /workspace

# 设置阿里云镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN npm config set registry https://registry.npmmirror.com

# 安装 pnpm
RUN npm install -g pnpm --registry=https://registry.npmmirror.com

# 构建前端
COPY frontend/package.json frontend/pnpm-lock.yaml ./frontend/
RUN cd frontend && pnpm install --frozen-lockfile
COPY frontend/ ./frontend/
RUN cd frontend && pnpm build-only

# 构建文档
COPY docs-site/ ./docs-site/
RUN cd docs-site && CI=true pnpm install --no-frozen-lockfile && pnpm build

# 第二阶段：构建 Go 应用
FROM golang:1.23-alpine AS go-builder

WORKDIR /build

ENV GOPROXY=https://goproxy.cn,direct
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache git gcc musl-dev sqlite-dev

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .
COPY --from=builder /workspace/frontend/dist ./frontend/dist
COPY --from=builder /workspace/dist ./dist

RUN BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S') && \
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") && \
    GIT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "dev") && \
    echo "Build info: $BUILD_DATE, $GIT_COMMIT, $GIT_VERSION" && \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-w -s -X 'main.buildDate=$BUILD_DATE' -X 'main.gitCommit=$GIT_COMMIT' -X 'main.gitVersion=$GIT_VERSION'" \
    -o cjrepo .

# 第三阶段：运行阶段
FROM alpine:latest

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates sqlite-libs

RUN addgroup -g 1000 cjrepo && \
    adduser -D -u 1000 -G cjrepo cjrepo

WORKDIR /app
COPY --from=go-builder /build/cjrepo .

RUN mkdir -p data storage && \
    chown -R cjrepo:cjrepo /app

USER cjrepo
EXPOSE 8060
CMD ["./cjrepo"]
