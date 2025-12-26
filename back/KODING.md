# Codebase Guidelines

## Build & Test Commands
- `go build`: Build the project
- `go test ./...`: Run all tests
- `go run main.go`: Start the server (main package)
- `go run server/server.go`: Start the Gin server
- `go run server2/server.go`: Start the HTTP server

## Code Style
- **Imports**: Group standard library, third-party, and local imports separately
- **Formatting**: Use `gofmt` for consistent formatting
- **Error Handling**: Always handle errors explicitly; avoid ignoring them
- **Naming**: Use camelCase for variables and functions, PascalCase for exported names
- **Logging**: Use `fmt.Println` for simple logging; consider structured logging for production

## Project Structure
- `main.go`: Primary server using Fiber
- `server/`: Gin-based server implementation
- `server2/`: Custom HTTP server with form parsing

## Dependencies
- `github.com/gofiber/fiber/v2`: Web framework
- `github.com/gin-gonic/gin`: Alternative web framework

# Codebase Guidelines

  ## 文件上传服务要求
  1. **严格配置**：
     - 监听地址：`:8060`
     - 路由路径：`/depot/publish/`
     - 文件保存位置：当前目录（使用用户上传的原文件名）

  2. **解析规则**：
     - 禁止使用 `bytes.Split` 或 `string` 转换
     - 必须使用游标式二进制解析
     - 使用结构体返回数据 

  3. 错误处理：
     - 无效的 `multipart/form-data` 格式返回 400
     - 文件保存失败返回 500

🤖 Generated with Anon Kode & deepseek-chat
