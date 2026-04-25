# 用户管理

## 管理后台

![用户管理](/assets/admin_users.png)

通过管理后台的「用户管理」页面可以：

- **创建用户** — 输入用户名和邮箱，系统自动生成 Token
- **启用/禁用** — 禁用后该用户无法发布和下载
- **重置 Token** — 重置后旧 Token 立即失效
- **删除用户** — 删除用户记录

## 命令行

```bash
# 添加用户
./cjrepo user add <username> <email>

# 列出用户
./cjrepo user list

# 删除用户
./cjrepo user delete <username>
```

## Token

每个用户拥有唯一的 Token，用于 cjpm 客户端的身份认证：

```bash
cjpm publish --token <user_token>
cjpm install <package_name> --token <user_token>
```

## 权限

用户权限由团队管理，详见 [团队权限](/guide/teams)。
