package middleware

import (
	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/auth"
	"ystyle.top/go/cjrepo/internal/models"
)

// JWTAuth 创建 JWT 认证中间件（用于管理后台）
func JWTAuth(authService *auth.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")

		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Authorization header is required",
			})
		}

		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid authorization format. Use: Bearer <token>",
			})
		}

		token := authHeader[7:]

		claims, err := authService.ValidateToken(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		if claims.UserType != "admin" {
			return c.Status(403).JSON(fiber.Map{
				"error": "Forbidden: admin access required",
			})
		}

		c.Locals("userType", claims.UserType)
		return c.Next()
	}
}

// TokenAuth 创建 Token 认证中间件（用于 cjpm 客户端）
func TokenAuth(engine *xorm.Engine) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")

		if token == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "authorization required",
			})
		}

		var user models.User
		has, err := engine.Where("token = ? AND is_active = ?", token, true).Get(&user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "database error",
			})
		}
		if !has {
			return c.Status(403).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		c.Locals("userID", user.ID)
		c.Locals("user", &user)
		return c.Next()
	}
}

// RequirePermission 权限检查中间件
func RequirePermission(engine *xorm.Engine, requiredPerm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(int64)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error": "authentication required",
			})
		}

		user, ok := c.Locals("user").(*models.User)
		if ok && user.IsSuperuser {
			return c.Next()
		}

		org := c.Query("organization", "")
		pkgName := c.Params("name")

		checker := auth.NewPermissionChecker(engine)
		if !checker.CheckPermission(userID, org, pkgName, requiredPerm) {
			return c.Status(403).JSON(fiber.Map{
				"error": "permission denied",
			})
		}

		return c.Next()
	}
}
