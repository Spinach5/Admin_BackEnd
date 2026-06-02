package main

import (
	"log"

	"web-backend/internal/config"
	"web-backend/internal/database"
	"web-backend/internal/handlers"
	"web-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	gin.SetMode(cfg.GinMode)

	database.Connect(cfg)
	defer database.Close()

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

			// 模块列表
			authorized.GET("/modules", handlers.GetModules())

			// 管理员管理 (超级管理员)
			admin := authorized.Group("/admin")
			admin.Use(middleware.RequireSuperAdmin())
			{
				admin.GET("/admins", handlers.GetAdmins())
				admin.POST("/admins", handlers.CreateAdmin())
				admin.PUT("/admins/:id", handlers.UpdateAdmin())
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

			// 事务种类管理
			authorized.GET("/affair-categories", handlers.GetAffairCategories())
			authorized.POST("/affair-categories", handlers.CreateAffairCategory())
			authorized.PUT("/affair-categories/:id", handlers.UpdateAffairCategory())
			authorized.DELETE("/affair-categories/:id", handlers.DeleteAffairCategory())


			// Excel 导入
			authorized.POST("/excel/import", handlers.ImportExcel())
			authorized.POST("/excel/preview", handlers.PreviewExcel())
		}

		// V1 普通用户接口
		v1 := r.Group("/api/v1")
		v1.Use(middleware.V1Auth())
		{
			v1.POST("/foods", handlers.V1GetFoods())
			v1.POST("/shops", handlers.V1GetShops())
			v1.POST("/affairs", handlers.V1GetAffairs())
			v1.POST("/books", handlers.V1GetBooks())
			v1.POST("/books/add", handlers.V1AddBook())
			v1.POST("/books/delete", handlers.V1DeleteBook())
		}
	}

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
