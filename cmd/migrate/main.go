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

	// 管理员表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS admins (
			id INT AUTO_INCREMENT PRIMARY KEY,
			account VARCHAR(50) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			is_super TINYINT(1) DEFAULT 0,
			is_active TINYINT(1) DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ admins")

		// 添加 last_active_at 列（兼容旧表）
		var adminsColCount int
		appDB.Get(&adminsColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'admins' AND COLUMN_NAME = 'last_active_at'")
		if adminsColCount == 0 {
			appDB.MustExec("ALTER TABLE admins ADD COLUMN last_active_at DATETIME DEFAULT NULL")
			log.Println("  ✓ admins.last_active_at 列已添加")
		}

	// 餐厅表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS shops (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			canteen_name VARCHAR(100) NOT NULL,
			school_id VARCHAR(50) NOT NULL DEFAULT 'hbut',
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
			school_id VARCHAR(50) NOT NULL DEFAULT 'hbut',
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
			school_id VARCHAR(50) NOT NULL DEFAULT 'hbut',
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

	// 普通用户表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS users (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',
			stuId VARCHAR(50) NOT NULL COMMENT '学号，唯一标识',
			nickName VARCHAR(100) NOT NULL COMMENT '昵称',
			schoolId VARCHAR(50) NOT NULL COMMENT '学校代码，关联学校信息表',
			password_hash VARCHAR(255) NOT NULL DEFAULT '' COMMENT '密码哈希',
			createdAt DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
			isDeleted TINYINT(1) UNSIGNED DEFAULT 0 COMMENT '软删除标记（0-未删除，1-已删除）'
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ users")

	// 添加 password_hash 列（兼容旧表）
	var colCount int
	appDB.Get(&colCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'password_hash'")
	if colCount == 0 {
		appDB.MustExec("ALTER TABLE users ADD COLUMN password_hash VARCHAR(255) NOT NULL DEFAULT ''")
		log.Println("  ✓ users.password_hash 列已添加")
	}

		// 添加 last_active_at 列（兼容旧表）
		var usersLastActiveColCount int
		appDB.Get(&usersLastActiveColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'last_active_at'")
		if usersLastActiveColCount == 0 {
			appDB.MustExec("ALTER TABLE users ADD COLUMN last_active_at DATETIME DEFAULT NULL")
			log.Println("  ✓ users.last_active_at 列已添加")
		}


		// 社团表
		appDB.MustExec(`
			CREATE TABLE IF NOT EXISTS clubs (
				id INT(11) AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(100) NOT NULL UNIQUE,
				introduction TEXT,
				activities TEXT,
				category VARCHAR(50),
				image_url VARCHAR(100),
				nature TINYINT(1) DEFAULT 0,
				contact VARCHAR(100)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
		`)
		log.Println("  ✓ clubs")

	// 书籍表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS book (
			book_id INT(10) UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '书籍ID',
			title VARCHAR(255) NOT NULL COMMENT '书名',
			category VARCHAR(100) COMMENT '分类',
			image_url VARCHAR(500) COMMENT '图片链接',
			price DECIMAL(10,2) COMMENT '价格',
			isbn VARCHAR(20) COMMENT 'ISBN',
			contact VARCHAR(255) COMMENT '联系方式',
			user_id INT(10) UNSIGNED NOT NULL COMMENT '发布者ID',
			status VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态',
			create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
			status_change_time DATETIME COMMENT '状态变更时间'
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)

	// 插入默认管理员账号
	var count int
	appDB.Get(&count, "SELECT COUNT(*) FROM admins WHERE account = 'admin'")
	if count == 0 {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 12)
		appDB.MustExec("INSERT INTO admins (account, password, is_super) VALUES (?, ?, ?)", "admin", string(hashed), 1)
		log.Println("已创建默认管理员: admin / admin123")
	} else {
		log.Println("管理员账号已存在，跳过")
	}

	log.Println("数据库迁移完成!")
}
