# 第一阶段：构建阶段
FROM golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /build

# 安装必要的构建工具
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# 复制 go.mod 和 go.sum（如果存在）
COPY go.mod go.sum* ./

# 下载依赖
RUN go mod download || true

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o cjrepo .

# 第二阶段：运行阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache ca-certificates sqlite-libs

# 创建非 root 用户
RUN addgroup -g 1000 cjrepo && \
    adduser -D -u 1000 -G cjrepo cjrepo

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/cjrepo .

# 创建必要的目录
RUN mkdir -p data storage && \
    chown -R cjrepo:cjrepo /app

# 切换到非 root 用户
USER cjrepo

# 暴露端口
EXPOSE 8060

# 设置默认命令
CMD ["./cjrepo"]
