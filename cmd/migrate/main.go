package main

import (
	"fmt"
	"log"

	"web-backend/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()

	// 先连接到 MySQL（不指定数据库），创建新数据库
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort)
	rootDB, err := sqlx.Open("mysql", rootDSN)
	if err != nil {
		log.Fatalf("连接 MySQL 失败: %v", err)
	}
	defer rootDB.Close()

	dbName := cfg.DBName

	// 创建数据库
	log.Printf("创建数据库: %s", dbName)
	_, err = rootDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName))
	if err != nil {
		log.Fatalf("创建数据库失败: %v", err)
	}
	rootDB.Close()

	// 连接到新数据库
	appDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, dbName)
	appDB, err := sqlx.Open("mysql", appDSN)
	if err != nil {
		log.Fatalf("连接新数据库失败: %v", err)
	}
	defer appDB.Close()

	log.Println("开始创建表...")

	// 用户表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			account VARCHAR(50) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			is_super TINYINT(1) DEFAULT 0,
			is_active TINYINT(1) DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ users")

	// 餐厅表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS shops (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			canteen_name VARCHAR(100) NOT NULL,
			rating DECIMAL(3,1) NOT NULL DEFAULT 0,
			comment TEXT NOT NULL,
			min DECIMAL(10,2) NOT NULL DEFAULT 0,
			max DECIMAL(10,2) NOT NULL DEFAULT 0
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ shops")

	// 食物表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS foods (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			shop_name VARCHAR(100) NOT NULL,
			canteen_name VARCHAR(100) NOT NULL,
			price DECIMAL(10,2) NOT NULL DEFAULT 0,
			taste VARCHAR(50) NOT NULL DEFAULT '',
			category VARCHAR(50) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ foods")

	// 事务表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS affairs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			category VARCHAR(100),
			link VARCHAR(500),
			details TEXT,
			channel VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ affairs")

	// 事务种类表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS affair_categories (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ affair_categories")

	// 插入默认管理员账号
	var count int
	appDB.Get(&count, "SELECT COUNT(*) FROM users WHERE account = 'admin'")
	if count == 0 {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 12)
		appDB.MustExec("INSERT INTO users (account, password, is_super) VALUES (?, ?, ?)", "admin", string(hashed), 1)
		log.Println("已创建默认管理员: admin / admin123")
	} else {
		log.Println("管理员账号已存在，跳过")
	}

	log.Println("数据库迁移完成!")
}
