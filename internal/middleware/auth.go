package middleware

import (
	"github.com/gofiber/fiber/v2"
	"ystyle.top/go/cjrepo/internal/auth"
)

// JWTAuth 创建 JWT 认证中间件
func JWTAuth(authService *auth.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 获取 Authorization header
		authHeader := c.Get("Authorization")

		// 检查 Bearer token
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Authorization header is required",
			})
		}

		// 验证 token 格式
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid authorization format. Use: Bearer <token>",
			})
		}

		token := authHeader[7:]

		// 验证 JWT token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// 验证用户类型
		if claims.UserType != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"error": "Forbidden: admin access required",
			})
		}

		// 认证成功，将用户信息存入 context
		c.Locals("userType", claims.UserType)
		return c.Next()
	}
}
