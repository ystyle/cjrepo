# 快速开始

## 系统要求

仓颉 SDK **v1.1.0** 或更高版本（[下载地址](https://cangjie-lang.cn/download)）。

## Docker Compose 一键启动（推荐）

```bash
git clone https://github.com/ystyle/cjrepo.git
cd cjrepo
docker-compose up -d --build
```

首次启动会自动初始化数据库和表结构。访问 http://localhost:8060 即可使用。

## 首次配置

1. 打开管理后台，用环境变量 `CJREPO_ADMIN_KEY` 的值登录
2. 创建用户（用于 cjpm 客户端发布/下载）
3. 创建组织（可选，用于包分组管理）
4. 配置团队（可选，用于细粒度权限控制）

## cjpm 客户端使用

cjpm 客户端通过 `cangjie-repo.toml` 配置仓库地址和 Token，配置方式详见[官方文档](https://pkgdocs.cangjie-lang.cn/docs/zh/1.0.0/central-repo/source_zh_cn/client/config.html)。

重点配置项：

```toml
[repository.home]
  registry = "http://localhost:8060"
  token = "<user_token>"
```

配置完成后：

```bash
cjpm publish   # 发布包
cjpm install   # 下载包
```

配置文件可在以下位置之一（按优先级）：

1. 项目内与 `cjpm.toml` 同级
2. `$HOME/.cjpm/cangjie-repo.toml`
3. 仓颉 SDK 的 `tools/config/cangjie-repo.toml`

然后使用 cjpm 命令：

```bash
# 发布包
cjpm publish

# 下载包
cjpm install <package_name>
```
