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

		// 添加 schoolId 列（兼容旧表）
		var adminsSchoolColCount int
		appDB.Get(&adminsSchoolColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'admins' AND COLUMN_NAME = 'schoolId'")
		if adminsSchoolColCount == 0 {
			appDB.MustExec("ALTER TABLE admins ADD COLUMN schoolId VARCHAR(50) NOT NULL DEFAULT ''")
			log.Println("  ✓ admins.schoolId 列已添加")
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

		// 添加 is_frozen 列（兼容旧表）
		var usersFrozenColCount int
		appDB.Get(&usersFrozenColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'is_frozen'")
		if usersFrozenColCount == 0 {
			appDB.MustExec("ALTER TABLE users ADD COLUMN is_frozen TINYINT(1) NOT NULL DEFAULT 0 COMMENT '冻结标记: 0正常 1冻结'")
			log.Println("  ✓ users.is_frozen 列已添加")
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

		// 添加 principal_id 列（兼容旧表）
		var clubsPrincipalColCount int
		appDB.Get(&clubsPrincipalColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'clubs' AND COLUMN_NAME = 'principal_id'")
		if clubsPrincipalColCount == 0 {
			appDB.MustExec("ALTER TABLE clubs ADD COLUMN principal_id INT(10) UNSIGNED DEFAULT NULL")
			appDB.MustExec("ALTER TABLE clubs ADD FOREIGN KEY (principal_id) REFERENCES users(id) ON DELETE SET NULL")
			log.Println("  ✓ clubs.principal_id 列已添加")
		} else {
			// 修复旧的外键约束：确保 ON DELETE SET NULL（旧版本可能创建了 RESTRICT）
			var clubsFkDeleteRule string
			appDB.Get(&clubsFkDeleteRule, "SELECT DELETE_RULE FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = 'clubs' AND CONSTRAINT_NAME = 'fk_clubs_principal'")
			if clubsFkDeleteRule != "SET NULL" {
				appDB.MustExec("ALTER TABLE clubs DROP FOREIGN KEY fk_clubs_principal")
				// 确保列可为 NULL
				appDB.MustExec("ALTER TABLE clubs MODIFY COLUMN principal_id INT(10) UNSIGNED DEFAULT NULL")
				appDB.MustExec("ALTER TABLE clubs ADD CONSTRAINT fk_clubs_principal FOREIGN KEY (principal_id) REFERENCES users(id) ON DELETE SET NULL")
				log.Println("  ✓ clubs.fk_clubs_principal 已修复为 ON DELETE SET NULL")
			}
		}

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

	// 书籍表 — 添加 description 列（兼容旧表）
	var bookDescColCount int
	appDB.Get(&bookDescColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'description'")
	if bookDescColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN description TEXT COMMENT '描述/详情'")
		log.Println("  ✓ book.description 列已添加")
	}

	// 书籍表 — 添加 condition 列（兼容旧表）
	var bookCondColCount int
	appDB.Get(&bookCondColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'condition'")
	if bookCondColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN `condition` VARCHAR(20) NOT NULL DEFAULT '几乎全新' COMMENT '新旧程度'")
		log.Println("  ✓ book.condition 列已添加")
	}

	// 书籍表 — 添加 school_id 列（兼容旧表）
	var bookSchoolColCount int
	appDB.Get(&bookSchoolColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'school_id'")
	if bookSchoolColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN school_id VARCHAR(50) NOT NULL DEFAULT 'hbut' COMMENT '学校代码'")
		log.Println("  ✓ book.school_id 列已添加")
	}

	// 书籍表 — 添加 is_delivery 列（兼容旧表）
	var bookIsDeliveryColCount int
	appDB.Get(&bookIsDeliveryColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'is_delivery'")
	if bookIsDeliveryColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN is_delivery TINYINT(1) NOT NULL DEFAULT 0 COMMENT '0自提 1帮送'")
		log.Println("  ✓ book.is_delivery 列已添加")
	}

	// 书籍表 — 添加 book_type 列（兼容旧表）
	var bookTypeColCount int
	appDB.Get(&bookTypeColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'book_type'")
	if bookTypeColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN book_type TINYINT NOT NULL DEFAULT 1 COMMENT '1卖书 2找书'")
		log.Println("  ✓ book.book_type 列已添加")
	}

	// 书籍表 — 添加 author 列（兼容旧表）
	var bookAuthorColCount int
	appDB.Get(&bookAuthorColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'author'")
	if bookAuthorColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN author VARCHAR(100) COMMENT '作者'")
		log.Println("  ✓ book.author 列已添加")
	}

	// 书籍表 — 添加 publisher 列（兼容旧表）
	var bookPublisherColCount int
	appDB.Get(&bookPublisherColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'publisher'")
	if bookPublisherColCount == 0 {
		appDB.MustExec("ALTER TABLE book ADD COLUMN publisher VARCHAR(100) COMMENT '出版社'")
		log.Println("  ✓ book.publisher 列已添加")
	}

	// 书籍表 — 删除 cover_url 列（回退，改用 image_url + book_images 首图作为封面）
	var bookCoverURLColCount int
	appDB.Get(&bookCoverURLColCount, "SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'book' AND COLUMN_NAME = 'cover_url'")
	if bookCoverURLColCount > 0 {
		appDB.MustExec("ALTER TABLE book DROP COLUMN cover_url")
		log.Println("  ✓ book.cover_url 列已删除")
	}

	// 书籍图片表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS book_images (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			book_id INT UNSIGNED NOT NULL,
			image_url VARCHAR(500) NOT NULL,
			sort_order TINYINT UNSIGNED DEFAULT 0,
			FOREIGN KEY (book_id) REFERENCES book(book_id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ book_images")

	// 书籍种类表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS book_categories (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			school_id VARCHAR(50) NOT NULL DEFAULT 'hbut',
			sort_order INT DEFAULT 0
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ book_categories")

	// 种子数据：默认书籍种类
	var catCount int
	appDB.Get(&catCount, "SELECT COUNT(*) FROM book_categories")
	if catCount == 0 {
		categories := []struct {
			Name      string
			SortOrder int
		}{
			{"数学", 1}, {"外语", 2}, {"计算机", 3}, {"理工类", 4},
			{"思政类", 5}, {"文学类", 6}, {"经管类", 7}, {"其他", 8},
		}
		for _, c := range categories {
			appDB.MustExec("INSERT INTO book_categories (name, school_id, sort_order) VALUES (?, 'hbut', ?)", c.Name, c.SortOrder)
		}
		log.Println("  ✓ book_categories 种子数据已插入")
	} else {
		log.Println("  book_categories 已有数据，跳过种子")
	}

	// 书籍想要表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS book_wants (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			book_id INT UNSIGNED NOT NULL,
			user_id INT UNSIGNED NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uk_book_user (book_id, user_id),
			FOREIGN KEY (book_id) REFERENCES book(book_id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ book_wants")

	// 聊天会话表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS conversation (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			book_id INT UNSIGNED NOT NULL,
			buyer_id INT UNSIGNED NOT NULL,
			seller_id INT UNSIGNED NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uk_book_buyer_seller (book_id, buyer_id, seller_id),
			FOREIGN KEY (book_id) REFERENCES book(book_id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ conversation")

	// 聊天消息表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS message (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			conversation_id INT UNSIGNED NOT NULL,
			sender_id INT UNSIGNED NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES conversation(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ message")

	// 购买记录表
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS purchase (
			purchase_id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			book_id INT UNSIGNED NOT NULL,
			buyer_id INT UNSIGNED NOT NULL,
			purchase_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (book_id) REFERENCES book(book_id),
			FOREIGN KEY (buyer_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	log.Println("  ✓ purchase")

	// 修复 purchase 外键约束（确保 ON DELETE CASCADE）
	var purchaseFkDeleteRule string
	appDB.Get(&purchaseFkDeleteRule, "SELECT DELETE_RULE FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = 'purchase' AND CONSTRAINT_NAME = 'fk_purchase_user'")
	if purchaseFkDeleteRule != "" && purchaseFkDeleteRule != "CASCADE" {
		appDB.MustExec("ALTER TABLE purchase DROP FOREIGN KEY fk_purchase_user")
		appDB.MustExec("ALTER TABLE purchase ADD CONSTRAINT fk_purchase_user FOREIGN KEY (buyer_id) REFERENCES users(id) ON DELETE CASCADE")
		log.Println("  ✓ purchase.fk_purchase_user 已修复为 ON DELETE CASCADE")
	}

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

	// ============ 教材管理相关表 ============
	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS materials (
			book_id     INT PRIMARY KEY AUTO_INCREMENT COMMENT '教材ID',
			isbn        VARCHAR(20) UNIQUE NOT NULL COMMENT '教材(ISBN)',
			title       VARCHAR(200) NOT NULL COMMENT '书名',
			author      VARCHAR(100) COMMENT '作者',
			publisher   VARCHAR(100) COMMENT '出版社',
			price       DECIMAL(10, 2) COMMENT '单价',
			created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
			extra_info  VARCHAR(100) COMMENT '备注信息',
			INDEX idx_isbn (isbn),
			INDEX idx_title (title)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='书籍信息表'
	`)
	log.Println("  ✓ materials")

	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS classes (
			class_id      INT PRIMARY KEY AUTO_INCREMENT COMMENT '班级ID',
			class_name    VARCHAR(50) NOT NULL COMMENT '班级名称，如"24软件工程3班"',
			grade         INT COMMENT '年级，如2024',
			major         VARCHAR(50) COMMENT '专业，如"软件工程"',
			department    VARCHAR(50) COMMENT '院系',
			student_count INT DEFAULT 0 COMMENT '班级人数',
			created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
			UNIQUE KEY uk_class_name (class_name),
			INDEX idx_grade_major (grade, major)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='班级信息表'
	`)
	log.Println("  ✓ classes")

	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS book_packages (
			package_id   INT PRIMARY KEY AUTO_INCREMENT COMMENT '教材包ID',
			package_name VARCHAR(100) NOT NULL COMMENT '教材包名称',
			grade        INT COMMENT '适用年级',
			major        VARCHAR(50) COMMENT '适用专业',
			semester     VARCHAR(20) COMMENT '学期',
			description  VARCHAR(255) COMMENT '描述',
			created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
			INDEX idx_grade_major_semester (grade, major, semester)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='教材包表'
	`)
	log.Println("  ✓ book_packages")

	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS package_books (
			id          INT PRIMARY KEY AUTO_INCREMENT COMMENT '明细ID',
			package_id  INT NOT NULL COMMENT '所属教材包ID',
			book_id     INT NOT NULL COMMENT '书籍ID',
			quantity    INT DEFAULT 1 COMMENT '数量',
			is_required BOOLEAN DEFAULT TRUE COMMENT '是否必修',
			FOREIGN KEY (package_id) REFERENCES book_packages(package_id) ON DELETE CASCADE,
			FOREIGN KEY (book_id) REFERENCES materials(book_id) ON DELETE CASCADE,
			UNIQUE KEY uk_package_book (package_id, book_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='教材包明细表'
	`)
	log.Println("  ✓ package_books")

	appDB.MustExec(`
		CREATE TABLE IF NOT EXISTS class_packages (
			id            INT PRIMARY KEY AUTO_INCREMENT COMMENT '关联ID',
			class_id      INT NOT NULL COMMENT '班级ID',
			package_id    INT NOT NULL COMMENT '教材包ID',
			academic_year VARCHAR(20) COMMENT '学年',
			FOREIGN KEY (class_id) REFERENCES classes(class_id) ON DELETE CASCADE,
			FOREIGN KEY (package_id) REFERENCES book_packages(package_id) ON DELETE CASCADE,
			UNIQUE KEY uk_class_package (class_id, package_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='班级-教材包关联表'
	`)
	log.Println("  ✓ class_packages")

	log.Println("数据库迁移完成!")
}
