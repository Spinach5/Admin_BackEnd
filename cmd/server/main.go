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
				admin.GET("/users", handlers.GetUsers())
				admin.POST("/users", handlers.CreateUser())
				admin.PUT("/users/:id", handlers.UpdateUser())
				admin.DELETE("/users/:id", handlers.DeleteUser())
			}

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

			// 课表查询
			authorized.POST("/classtable", handlers.GetClasstable())

			// Excel 导入
			authorized.POST("/excel/import", handlers.ImportExcel())
			authorized.POST("/excel/preview", handlers.PreviewExcel())
		}
	}

	log.Printf("服务器运行在：http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func resetLoginStatus() {
	_, err := database.DB.Exec("UPDATE users SET is_active = 0 WHERE is_active = 1")
	if err != nil {
		log.Printf("重置登录状态失败: %v", err)
		return
	}
	log.Println("已重置所有用户登录状态")
}
