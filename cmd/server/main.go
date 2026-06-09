package main

import (
	"log"
	"os"
	"time"

	"web-backend/internal/config"
	"web-backend/internal/database"
	"web-backend/internal/handlers"
	"web-backend/internal/middleware"
	"web-backend/internal/models"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	gin.SetMode(cfg.GinMode)

	database.Connect(cfg)
	defer database.Close()

	if _, err := os.Stat(cfg.UploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
			log.Fatalf("创建上传目录失败 %s: %v", cfg.UploadDir, err)
		}
	}

	resetLoginStatus()

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg))

	api := r.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)

		// 认证路由 (无需登录)
		auth := api.Group("/auth")
		{
			auth.POST("/login", handlers.Login(cfg))
		}

		// 需要认证的路由
		authorized := api.Group("")
		authorized.Use(middleware.Auth(cfg))
		{
			authorized.POST("/auth/logout", handlers.Logout())
			authorized.GET("/auth/me", handlers.GetMe())
			authorized.PUT("/auth/change-password", handlers.ChangePassword())
			authorized.POST("/auth/heartbeat", handlers.Heartbeat())

			// 模块列表
			authorized.GET("/modules", handlers.GetModules())

			// 管理员管理 (超级管理员)
			admin := authorized.Group("/admin")
			admin.Use(middleware.RequireSuperAdmin())
			{
				admin.GET("/admins", handlers.GetAdmins())
				admin.POST("/admins", handlers.CreateAdmin())
				admin.PUT("/admins/:id", handlers.UpdateAdmin())
				admin.PUT("/admins/:id/info", handlers.UpdateAdminInfo())
				admin.DELETE("/admins/:id", handlers.DeleteAdmin())
			}

			// 普通用户管理
			authorized.GET("/users", handlers.GetUsers())
			authorized.GET("/users/:id", handlers.GetUserByID())
			authorized.POST("/users", handlers.CreateUser())
			authorized.PUT("/users/:id", handlers.UpdateUser())
			authorized.DELETE("/users/:id", handlers.SoftDeleteUser())
			authorized.DELETE("/users/:id/hard", handlers.HardDeleteUser())

			// 书籍管理
			authorized.GET("/books/categories", handlers.GetBookCategories())
			authorized.GET("/books", handlers.GetBooks())
			authorized.GET("/books/:id", handlers.GetBookByID())
			authorized.POST("/books", handlers.CreateBook())
			authorized.PUT("/books/:id", handlers.UpdateBook())
			authorized.DELETE("/books/:id", handlers.DeleteBook())

			// 餐厅管理
			authorized.GET("/shops", handlers.GetShops())
			authorized.POST("/shops", handlers.CreateShop())
			authorized.PUT("/shops/:id", handlers.UpdateShop())
			authorized.DELETE("/shops/:id", handlers.DeleteShop())

			// 食物管理
			authorized.GET("/foods", handlers.GetFoods())
			authorized.POST("/foods", handlers.CreateFood())
			authorized.PUT("/foods/:id", handlers.UpdateFood())
			authorized.DELETE("/foods/:id", handlers.DeleteFood())

			// 事务管理
			authorized.GET("/affairs", handlers.GetAffairs())
			authorized.POST("/affairs", handlers.CreateAffair())
			authorized.PUT("/affairs/:id", handlers.UpdateAffair())
			authorized.DELETE("/affairs/:id", handlers.DeleteAffair())

			// 社团管理
			authorized.GET("/clubs", handlers.GetClubs())
			authorized.GET("/clubs/:id", handlers.GetClubByID())
			authorized.POST("/clubs", handlers.CreateClub())
			authorized.PUT("/clubs/:id", handlers.UpdateClub())
			authorized.DELETE("/clubs/:id", handlers.DeleteClub())

			// 事务种类管理
			authorized.GET("/affair-categories", handlers.GetAffairCategories())
			authorized.POST("/affair-categories", handlers.CreateAffairCategory())
			authorized.PUT("/affair-categories/:id", handlers.UpdateAffairCategory())
			authorized.DELETE("/affair-categories/:id", handlers.DeleteAffairCategory())

			// Excel 导入
			authorized.POST("/excel/import", handlers.ImportExcel())
			authorized.POST("/excel/preview", handlers.PreviewExcel())
		}

		// V1 学生接口 (JWT)
		v1 := r.Group("/api/v1")
		{
			v1.POST("/auth/register", handlers.StudentRegister(cfg))
			v1.POST("/auth/login", handlers.StudentLogin(cfg))
			v1.GET("/auth/check-user", handlers.CheckUser())

			v1Auth := v1.Group("")
			v1Auth.Use(middleware.StudentAuth(cfg))
			{
				v1Auth.GET("/auth/me", handlers.StudentMe())
				v1Auth.POST("/auth/heartbeat", handlers.StudentHeartbeat())
				v1Auth.GET("/clubs", handlers.V1GetClubs())
				v1Auth.GET("/clubs/:id", handlers.V1GetClubByID())
				v1Auth.POST("/clubs", handlers.V1CreateClub())
				v1Auth.GET("/books/categories", handlers.GetBookCategories())
				v1Auth.GET("/books", handlers.V1GetBooks())
				v1Auth.GET("/books/mine", handlers.V1GetMyBooks())
				v1Auth.GET("/books/:id", handlers.V1GetBookByID())
				v1Auth.POST("/books", handlers.V1CreateBook())
				v1Auth.PUT("/books/:id", handlers.V1UpdateBook())
				v1Auth.DELETE("/books/:id", handlers.V1DeleteBook())
			}
		}
	}

	r.Static("/uploads", cfg.UploadDir)

	// 后台定时清理过期管理员会话
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if err := models.CleanStaleAdminSessions(database.DB, 5); err != nil {
				log.Printf("清理过期会话失败: %v", err)
			}
		}
	}()

	log.Printf("服务器运行在：http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func resetLoginStatus() {
	_, err := database.DB.Exec("UPDATE admins SET is_active = 0 WHERE is_active = 1")
	if err != nil {
		log.Printf("重置登录状态失败: %v", err)
		return
	}
	log.Println("已重置所有用户登录状态")
}
