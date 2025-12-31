package main

import (
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/auth"
	handlers "ystyle.top/go/cjrepo/internal/handlers"
	"ystyle.top/go/cjrepo/internal/middleware"
	"ystyle.top/go/cjrepo/internal/models"
	"ystyle.top/go/cjrepo/internal/storage"
	upstream2 "ystyle.top/go/cjrepo/internal/upstream"
)

const (
	dbPath      = "./data/cjrepo.db"
	storagePath = "./storage"
	defaultPort = ":8060"
)

func main() {
	if len(os.Args) > 1 {
		// Handle subcommands
		switch os.Args[1] {
		case "user":
			handleUserCommand()
			return
		case "help", "-h", "--help":
			printHelp()
			return
		case "version", "-v", "--version":
			printVersion()
			return
		}
	}

	// Start server by default
	startServer()
}

func printHelp() {
	fmt.Println("CJRepo - 仓颉中央库服务")
	fmt.Println("\n用法:")
	fmt.Println("  cjrepo [command] [arguments]")
	fmt.Println("\n命令:")
	fmt.Println("  server        启动服务器（默认）")
	fmt.Println("  user         用户管理")
	fmt.Println("  help, -h     显示帮助信息")
	fmt.Println("  version, -v  显示版本信息")
	fmt.Println("\n用户管理:")
	fmt.Println("  cjrepo user add <username> <email>      创建新用户")
	fmt.Println("  cjrepo user list                        列出所有用户")
	fmt.Println("  cjrepo user delete <username>            删除用户")
	fmt.Println("\n示例:")
	fmt.Println("  cjrepo                    # 启动服务器")
	fmt.Println("  cjrepo user add alice alice@example.com")
	fmt.Println("  cjrepo user list")
	fmt.Println("  cjrepo user delete bob")
}

func printVersion() {
	fmt.Println("CJRepo v1.0.0 - 仓颉中央库服务")
}

func startServer() {
	// Check for admin key（管理密钥，不是 JWT token）
	adminKey := os.Getenv("CJREPO_ADMIN_KEY")
	if adminKey == "" {
		log.Fatal("CJREPO_ADMIN_KEY environment variable is required")
	}
	log.Println("Admin key configured")

	// Check if require auth for download and index
	requireAuth := os.Getenv("CJREPO_REQUIRE_AUTH") == "true"
	if requireAuth {
		log.Println("[INFO] REQUIRE_AUTH is ENABLED: download and index requests require token")
	} else {
		log.Println("[INFO] REQUIRE_AUTH is DISABLED: download and index requests are public")
	}

	// Get default organization
	defaultOrg := os.Getenv("CJREPO_DEFAULT_ORGANIZATION")
	if defaultOrg != "" {
		log.Printf("[INFO] Default organization: %s", defaultOrg)
	} else {
		log.Println("[INFO] No default organization configured")
	}

	// 1. Initialize database
	engine, err := initDatabase(dbPath, defaultOrg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 2. Initialize storage manager
	storageMgr := storage.NewStorageManager(storagePath)

	// 3. Initialize upstream sync
	upstreamSync := upstream2.NewSync(engine)

	// 4. Initialize auth service
	authService := auth.NewAuthService(adminKey)

	// 5. Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})
	app.Use(logger.New())
	app.Use(recover.New())

	// 6. Register routes
	setupRoutes(app, engine, storageMgr, authService, upstreamSync, requireAuth)

	// 6. Start server
	log.Printf("Cangjie Depot Server starting on %s", defaultPort)
	log.Fatal(app.Listen(defaultPort))
}

// initDatabase initializes the database connection and creates tables
func initDatabase(path string, defaultOrg string) (*xorm.Engine, error) {
	engine, err := xorm.NewEngine("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Show SQL logging
	engine.ShowSQL(true)

	// Sync table structures
	if err := engine.Sync2(
		new(models.Package),
		new(models.User),
		new(models.PublishLog),
		new(models.AdminLog),
		new(models.Upstream),
		new(models.Organization),
		new(models.OrganizationMember),
	); err != nil {
		return nil, fmt.Errorf("failed to sync database: %w", err)
	}

	// Create default organization if specified
	if defaultOrg != "" {
		var existing models.Organization
		has, err := engine.Where("name = ?", defaultOrg).Get(&existing)
		if err != nil {
			log.Printf("[ERROR] Failed to check default organization: %v", err)
		} else if !has {
			// Create default organization
			org := &models.Organization{
				Name:        defaultOrg,
				DisplayName: defaultOrg,
				Description: "Default organization",
				IsDefault:   true,
			}
			if _, err := engine.Insert(org); err != nil {
				log.Printf("[ERROR] Failed to create default organization: %v", err)
			} else {
				log.Printf("[INFO] Created default organization: %s", defaultOrg)
			}
		} else {
			// Update to default if not already
			if !existing.IsDefault {
				existing.IsDefault = true
				engine.ID(existing.ID).Update(&existing)
				log.Printf("[INFO] Set existing organization as default: %s", defaultOrg)
			}
		}
	}

	log.Println("Database initialized successfully")
	return engine, nil
}

// setupRoutes configures all application routes
func setupRoutes(app *fiber.App, engine *xorm.Engine, storageMgr *storage.Manager, authService *auth.AuthService, upstreamSync *upstream2.Sync, requireAuth bool) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// cjpm protocol routes
	publishHandler := handlers.NewPublishHandler(engine, storageMgr)
	downloadHandler := handlers.NewDownloadHandler(engine, upstreamSync, requireAuth)
	indexHandler := handlers.NewIndexHandler(engine, upstreamSync, requireAuth)

	// Publish endpoint: POST /pkg/{name}?organization={org}
	app.Post("/pkg/:name", publishHandler.HandlePublish)

	// Download endpoint: GET /pkg/{name}/{version}?organization={org}
	app.Get("/pkg/:name/:version", downloadHandler.HandleDownload)

	// Index endpoint: GET /index/{first}/{second}/{name}?organization={org}
	app.Get("/index/:first/:second/:name", indexHandler.HandleIndex)

	// Legacy routes (for backward compatibility)
	depot := app.Group("/depot")
	depot.Post("/list/all", legacyHandler)
	depot.Post("/list", legacyHandler)
	depot.Post("/download", legacyHandler)
	depot.Post("/publish", legacyHandler)

	// Public API routes
	publicHandler := handlers.NewPublicHandler(engine)
	app.Get("/api/stats", publicHandler.GetStats)
	app.Get("/api/packages", publicHandler.ListPackages)
	app.Get("/api/packages/:name", publicHandler.GetPackage)
	app.Get("/api/packages/:name/:version", publicHandler.GetPackageVersion)
	app.Get("/api/organizations", publicHandler.GetOrganizations)

	// Admin API routes
	adminHandler := handlers.NewAdminHandler(engine, authService, storageMgr)
	jwtAuth := middleware.JWTAuth(authService)

	admin := app.Group("/api/admin")

	// 登录端点（无需认证）
	admin.Post("/login", adminHandler.Login)

	// 需要 JWT 认证的路由
	admin.Use(jwtAuth)
	admin.Get("/dashboard", adminHandler.GetDashboardStats)
	admin.Get("/stats", adminHandler.GetDashboardStats) // Alias for dashboard

	// Package management
	admin.Get("/packages", adminHandler.ListPackages)
	admin.Get("/packages/versions/:name", adminHandler.GetPackageVersions)
	admin.Delete("/packages/:id", adminHandler.DeletePackage)
	admin.Put("/packages/:id/restore", adminHandler.RestorePackage)
	admin.Delete("/packages/:id/hard", adminHandler.HardDeletePackage)

	// User management
	admin.Get("/users", adminHandler.ListUsers)
	admin.Post("/users", adminHandler.CreateUser)
	admin.Delete("/users/:id", adminHandler.DeleteUser)
	admin.Put("/users/:id/toggle", adminHandler.ToggleUser)
	admin.Post("/users/:id/reset-token", adminHandler.ResetUserToken)

	// Logs
	admin.Get("/logs/publish", adminHandler.GetPublishLogs)
	admin.Get("/logs/admin", adminHandler.GetAdminLogs)

	// Upstream management
	upstreamHandler := handlers.NewUpstreamHandler(engine)
	admin.Get("/upstreams", upstreamHandler.ListUpstreams)
	admin.Post("/upstreams", upstreamHandler.CreateUpstream)
	admin.Put("/upstreams/:id", upstreamHandler.UpdateUpstream)
	admin.Delete("/upstreams/:id", upstreamHandler.DeleteUpstream)
	admin.Post("/upstreams/:id/test", upstreamHandler.TestUpstream)
	admin.Get("/upstreams/:id/cache-stats", upstreamHandler.GetUpstreamCacheStats)
	admin.Post("/upstreams/:id/clear-cache", upstreamHandler.ClearUpstreamCache)

	// Organization management
	organizationHandler := handlers.NewOrganizationHandler(engine)
	admin.Get("/organizations", organizationHandler.ListOrganizations)
	admin.Post("/organizations", organizationHandler.CreateOrganization)
	admin.Put("/organizations/:id", organizationHandler.UpdateOrganization)
	admin.Delete("/organizations/:id", organizationHandler.DeleteOrganization)
	admin.Get("/organizations/:id/members", organizationHandler.GetOrganizationMembers)
	admin.Post("/organizations/:id/members", organizationHandler.AddMember)
	admin.Delete("/organizations/:id/members/:user_id", organizationHandler.RemoveMember)

	// SPA fallback - handle all non-API routes
	app.All("/*", func(c *fiber.Ctx) error {
		// Skip API routes
		requestPath := c.Path()
		if len(requestPath) > 0 {
			// Skip cjpm protocol routes
			if requestPath == "/health" || strings.HasPrefix(requestPath, "/pkg/") || strings.HasPrefix(requestPath, "/index/") {
				return c.Status(404).SendString("Not found")
			}
			// Skip admin API routes
			if strings.HasPrefix(requestPath, "/api/") {
				return c.Status(404).SendString("Not found")
			}
			// Skip legacy depot routes
			if strings.HasPrefix(requestPath, "/depot/") {
				return c.Status(404).SendString("Not found")
			}
		}

		// Try to read file from embedded filesystem
		filePath := "frontend/dist" + requestPath

		// Read file
		file, err := webFS.ReadFile(filePath)
		if err != nil {
			// File not found, fallback to index.html for SPA
			file, err = webFS.ReadFile("frontend/dist/index.html")
			if err != nil {
				return c.Status(404).SendString("Not found")
			}
			filePath = "frontend/dist/index.html"
		}

		// Set content type using mime package
		ext := filepath.Ext(filePath)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Set("Content-Type", contentType)

		// Write response
		_, err = c.Write(file)
		return err
	})

	log.Println("Routes registered successfully")
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}

// legacyHandler handles legacy depot routes
func legacyHandler(c *fiber.Ctx) error {
	log.Printf("Legacy endpoint called: %s %s", c.Method(), c.Path())
	return c.SendString("ok")
}

// handleUserCommand handles user management subcommands
func handleUserCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Error: user command requires an action")
		fmt.Println("\nUsage:")
		fmt.Println("  cjrepo user add <username> <email>     Create a new user")
		fmt.Println("  cjrepo user list                        List all users")
		fmt.Println("  cjrepo user delete <username>            Delete a user")
		os.Exit(1)
	}

	action := os.Args[2]

	switch action {
	case "add":
		addUser()
	case "list":
		listUsers()
	case "delete":
		deleteUser()
	default:
		fmt.Printf("Error: unknown user action '%s'\n", action)
		fmt.Println("\nAvailable actions: add, list, delete")
		os.Exit(1)
	}
}

// addUser creates a new user
func addUser() {
	if len(os.Args) < 5 {
		fmt.Println("Error: username and email are required")
		fmt.Println("Usage: cjrepo user add <username> <email>")
		os.Exit(1)
	}

	username := os.Args[3]
	email := os.Args[4]

	// Connect to database
	engine, err := xorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer engine.Close()

	// Check if user already exists
	exists, _ := engine.Where("username = ?", username).Exist(&models.User{})
	if exists {
		fmt.Printf("Error: user '%s' already exists\n", username)
		os.Exit(1)
	}

	// Generate token
	token := fmt.Sprintf("token-%s-%d", username, os.Getpid())

	// Create user
	user := &models.User{
		Username: username,
		Token:    token,
		Email:    email,
		IsActive: true,
	}

	if _, err := engine.Insert(user); err != nil {
		log.Fatal("Failed to create user:", err)
	}

	fmt.Println("✓ User created successfully!")
	fmt.Println("\nUser Information:")
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  Email: %s\n", email)
	fmt.Printf("  Token: %s\n", token)
	fmt.Println("\nAdd this to your cangjie-repo.toml or ~/.cjr/config.toml:")
	fmt.Printf("  [repository.home]\n")
	fmt.Printf("  registry = \"http://localhost:8060\"\n")
	fmt.Printf("  token = \"%s\"\n", token)
}

// listUsers lists all users
func listUsers() {
	// Connect to database
	engine, err := xorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer engine.Close()

	// Query all users
	var users []models.User
	if err := engine.Find(&users); err != nil {
		log.Fatal("Failed to query users:", err)
	}

	if len(users) == 0 {
		fmt.Println("No users found")
		return
	}

	fmt.Println("\nUsers:")
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Printf("%-15s %-30s %-30s %-10s\n", "Username", "Email", "Token", "Status")
	fmt.Println("─────────────────────────────────────────────────────────────")

	for _, user := range users {
		status := "✓ Active"
		if !user.IsActive {
			status = "✗ Inactive"
		}
		fmt.Printf("%-15s %-30s %-30s %-10s\n",
			user.Username,
			user.Email,
			user.Token,
			status)
	}
	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Printf("Total: %d user(s)\n", len(users))
}

// deleteUser deletes a user
func deleteUser() {
	if len(os.Args) < 4 {
		fmt.Println("Error: username is required")
		fmt.Println("Usage: cjrepo user delete <username>")
		os.Exit(1)
	}

	username := os.Args[3]

	// Connect to database
	engine, err := xorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer engine.Close()

	// Check if user exists
	var user models.User
	has, err := engine.Where("username = ?", username).Get(&user)
	if err != nil {
		log.Fatal("Database error:", err)
	}

	if !has {
		fmt.Printf("Error: user '%s' not found\n", username)
		os.Exit(1)
	}

	// Delete user
	_, err = engine.Delete(&user)
	if err != nil {
		log.Fatal("Failed to delete user:", err)
	}

	fmt.Printf("✓ User '%s' deleted successfully\n", username)
}
