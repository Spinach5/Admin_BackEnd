package models

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Material 教材信息
type Material struct {
	BookID    int             `db:"book_id" json:"book_id"`
	ISBN      string          `db:"isbn" json:"isbn"`
	Title     string          `db:"title" json:"title"`
	Author    sql.NullString  `db:"author" json:"author"`
	Publisher sql.NullString  `db:"publisher" json:"publisher"`
	Price     sql.NullFloat64 `db:"price" json:"price"`
	CreatedAt sql.NullTime    `db:"created_at" json:"created_at"`
	ExtraInfo sql.NullString  `db:"extra_info" json:"extra_info"`
}

// Class 班级信息
type Class struct {
	ClassID      int            `db:"class_id" json:"class_id"`
	ClassName    string         `db:"class_name" json:"class_name"`
	Grade        sql.NullInt64  `db:"grade" json:"grade"`
	Major        sql.NullString `db:"major" json:"major"`
	Department   sql.NullString `db:"department" json:"department"`
	StudentCount int            `db:"student_count" json:"student_count"`
	CreatedAt    sql.NullTime   `db:"created_at" json:"created_at"`
}

// BookPackage 教材包
type BookPackage struct {
	PackageID   int            `db:"package_id" json:"package_id"`
	PackageName string         `db:"package_name" json:"package_name"`
	Grade       sql.NullInt64  `db:"grade" json:"grade"`
	Major       sql.NullString `db:"major" json:"major"`
	Semester    sql.NullString `db:"semester" json:"semester"`
	Description sql.NullString `db:"description" json:"description"`
	CreatedAt   sql.NullTime   `db:"created_at" json:"created_at"`
}

// PackageBook 教材包明细
type PackageBook struct {
	ID         int  `db:"id" json:"id"`
	PackageID  int  `db:"package_id" json:"package_id"`
	BookID     int  `db:"book_id" json:"book_id"`
	Quantity   int  `db:"quantity" json:"quantity"`
	IsRequired bool `db:"is_required" json:"is_required"`
}

// ClassPackage 班级-教材包关联
type ClassPackage struct {
	ID           int            `db:"id" json:"id"`
	ClassID      int            `db:"class_id" json:"class_id"`
	PackageID    int            `db:"package_id" json:"package_id"`
	AcademicYear sql.NullString `db:"academic_year" json:"academic_year"`
}

// MaterialDetail 教材详情（含教材包信息）
type MaterialDetail struct {
	Material
	CourseName string `json:"course_name,omitempty"`
	PackageID  int    `json:"package_id,omitempty"`
	Quantity   int    `json:"quantity,omitempty"`
}

// ============ materials CRUD ============

func GetAllMaterials(db *sqlx.DB) ([]Material, error) {
	materials := make([]Material, 0)
	err := db.Select(&materials, `SELECT book_id, isbn, title, author, publisher, price, created_at, extra_info
		FROM materials ORDER BY book_id DESC`)
	return materials, err
}

func GetMaterialByID(db *sqlx.DB, id int) (*Material, error) {
	var m Material
	err := db.Get(&m, `SELECT book_id, isbn, title, author, publisher, price, created_at, extra_info
		FROM materials WHERE book_id = ?`, id)
	return &m, err
}

func GetMaterialByISBN(db *sqlx.DB, isbn string) (*Material, error) {
	var m Material
	err := db.Get(&m, `SELECT book_id, isbn, title, author, publisher, price, created_at, extra_info
		FROM materials WHERE isbn = ?`, isbn)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func CreateMaterial(db *sqlx.DB, m *Material) error {
	result, err := db.Exec(`INSERT INTO materials (isbn, title, author, publisher, price, extra_info)
		VALUES (?, ?, ?, ?, ?, ?)`,
		m.ISBN, m.Title, m.Author, m.Publisher, m.Price, m.ExtraInfo)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	m.BookID = int(id)
	return nil
}

func UpdateMaterial(db *sqlx.DB, m *Material) error {
	_, err := db.Exec(`UPDATE materials SET isbn=?, title=?, author=?, publisher=?, price=?, extra_info=?
		WHERE book_id=?`,
		m.ISBN, m.Title, m.Author, m.Publisher, m.Price, m.ExtraInfo, m.BookID)
	return err
}

func DeleteMaterial(db *sqlx.DB, id int) error {
	_, err := db.Exec("DELETE FROM materials WHERE book_id = ?", id)
	return err
}

// SearchMaterials 支持按 ISBN/书名 关键字搜索
func SearchMaterials(db *sqlx.DB, keyword string) ([]Material, error) {
	materials := make([]Material, 0)
	if keyword == "" {
		return materials, nil
	}
	like := "%" + keyword + "%"
	err := db.Select(&materials, `SELECT book_id, isbn, title, author, publisher, price, created_at, extra_info
		FROM materials
		WHERE isbn LIKE ? OR title LIKE ? OR author LIKE ? OR publisher LIKE ?
		ORDER BY book_id DESC`, like, like, like, like)
	return materials, err
}

// ============ classes ============

// ParseClassName 从班级名称中解析年级和专业，例如 "24软件工程3班" => (2024, "软件工程")
func ParseClassName(className string) (grade int, major string) {
	className = strings.TrimSpace(className)
	if className == "" {
		return 0, ""
	}

	// 匹配两位数字开头的年级
	re := regexp.MustCompile(`^(\d{2,4})\s*(.+?)(\d+)\s*班?$`)
	m := re.FindStringSubmatch(className)
	if m != nil {
		yearStr := m[1]
		switch len(yearStr) {
		case 2:
			// 24 => 2024
			y, err := strconv.Atoi(yearStr)
			if err == nil {
				if y >= 50 {
					grade = 1900 + y
				} else {
					grade = 2000 + y
				}
			}
		case 4:
			y, err := strconv.Atoi(yearStr)
			if err == nil {
				grade = y
			}
		}
		major = strings.TrimSpace(m[2])
		return grade, major
	}

	// 仅专业无班号
	re2 := regexp.MustCompile(`^(\d{2,4})\s*(.+)$`)
	m2 := re2.FindStringSubmatch(className)
	if m2 != nil {
		yearStr := m2[1]
		y, err := strconv.Atoi(yearStr)
		if err == nil {
			switch len(yearStr) {
			case 2:
				if y >= 50 {
					grade = 1900 + y
				} else {
					grade = 2000 + y
				}
			case 4:
				grade = y
			}
		}
		major = strings.TrimSpace(m2[2])
	}

	return grade, major
}

// GetOrCreateClass 按班级名称查找或创建班级记录
func GetOrCreateClass(db *sqlx.DB, className, department string) (*Class, error) {
	className = strings.TrimSpace(className)
	if className == "" {
		return nil, errors.New("班级名称不能为空")
	}

	var c Class
	err := db.Get(&c, `SELECT class_id, class_name, grade, major, department, student_count, created_at
		FROM classes WHERE class_name = ?`, className)
	if err == nil {
		// 若已存在但 department 为空，则补全
		if department != "" && !c.Department.Valid {
			db.Exec("UPDATE classes SET department = ? WHERE class_id = ?", department, c.ClassID)
			c.Department = sql.NullString{String: department, Valid: true}
		}
		return &c, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	grade, major := ParseClassName(className)
	result, err := db.Exec(`INSERT INTO classes (class_name, grade, major, department, student_count)
		VALUES (?, ?, ?, ?, 0)`,
		className,
		sql.NullInt64{Int64: int64(grade), Valid: grade > 0},
		sql.NullString{String: major, Valid: major != ""},
		sql.NullString{String: department, Valid: department != ""},
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &Class{
		ClassID:    int(id),
		ClassName:  className,
		Grade:      sql.NullInt64{Int64: int64(grade), Valid: grade > 0},
		Major:      sql.NullString{String: major, Valid: major != ""},
		Department: sql.NullString{String: department, Valid: department != ""},
	}, nil
}

func GetAllClasses(db *sqlx.DB) ([]Class, error) {
	classes := make([]Class, 0)
	err := db.Select(&classes, `SELECT class_id, class_name, grade, major, department, student_count, created_at
		FROM classes ORDER BY class_id`)
	return classes, err
}

// ============ book_packages ============

// GetOrCreatePackage 按学年+年级+专业+学期查找或创建教材包
func GetOrCreatePackage(db *sqlx.DB, grade int, major, semester, academicYear string) (*BookPackage, error) {
	if semester == "" {
		return nil, errors.New("学期不能为空")
	}

	// 先按 (grade, major, semester) 查找
	var pkg BookPackage
	query := `SELECT package_id, package_name, grade, major, semester, description, created_at
		FROM book_packages WHERE semester = ?`
	args := []interface{}{semester}
	if grade > 0 {
		query += " AND grade = ?"
		args = append(args, grade)
	} else {
		query += " AND grade IS NULL"
	}
	if major != "" {
		query += " AND major = ?"
		args = append(args, major)
	} else {
		query += " AND major IS NULL"
	}
	query += " LIMIT 1"

	err := db.Get(&pkg, query, args...)
	if err == nil {
		return &pkg, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// 构造包名
	pkgName := buildPackageName(grade, major, semester, academicYear)
	desc := academicYear
	result, err := db.Exec(`INSERT INTO book_packages (package_name, grade, major, semester, description)
		VALUES (?, ?, ?, ?, ?)`,
		pkgName,
		sql.NullInt64{Int64: int64(grade), Valid: grade > 0},
		sql.NullString{String: major, Valid: major != ""},
		sql.NullString{String: semester, Valid: semester != ""},
		sql.NullString{String: desc, Valid: desc != ""},
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return &BookPackage{
		PackageID:   int(id),
		PackageName: pkgName,
		Grade:       sql.NullInt64{Int64: int64(grade), Valid: grade > 0},
		Major:       sql.NullString{String: major, Valid: major != ""},
		Semester:    sql.NullString{String: semester, Valid: semester != ""},
		Description: sql.NullString{String: desc, Valid: desc != ""},
	}, nil
}

func buildPackageName(grade int, major, semester, academicYear string) string {
	yearPart := ""
	if academicYear != "" {
		yearPart = academicYear + " "
	}
	gradePart := ""
	if grade > 0 {
		gradePart = strconv.Itoa(grade) + "级"
	}
	majorPart := ""
	if major != "" {
		majorPart = major
	}
	semPart := semester
	if semPart == "" {
		semPart = "未指定学期"
	}
	return strings.TrimSpace(yearPart + gradePart + majorPart + "-" + semPart)
}

// AddBookToPackage 将书加入教材包（已存在则跳过）
func AddBookToPackage(db *sqlx.DB, packageID, bookID, quantity int, isRequired bool) error {
	_, err := db.Exec(`INSERT IGNORE INTO package_books (package_id, book_id, quantity, is_required)
		VALUES (?, ?, ?, ?)`, packageID, bookID, quantity, isRequired)
	return err
}

// LinkClassPackage 关联班级与教材包（已存在则跳过）
func LinkClassPackage(db *sqlx.DB, classID, packageID int, academicYear string) error {
	_, err := db.Exec(`INSERT IGNORE INTO class_packages (class_id, package_id, academic_year)
		VALUES (?, ?, ?)`, classID, packageID, sql.NullString{String: academicYear, Valid: academicYear != ""})
	return err
}

// ============ 查询接口 ============

// GetMaterialsByClassAndSemester 按班级和学期查询教材
func GetMaterialsByClassAndSemester(db *sqlx.DB, className, semester string) ([]MaterialDetail, error) {
	details := make([]MaterialDetail, 0)
	if className == "" || semester == "" {
		return details, nil
	}

	query := `SELECT m.book_id, m.isbn, m.title, m.author, m.publisher, m.price, m.created_at, m.extra_info,
			pb.quantity AS quantity,
			p.package_id AS package_id
		FROM class_packages cp
		JOIN classes c ON cp.class_id = c.class_id
		JOIN book_packages p ON cp.package_id = p.package_id
		JOIN package_books pb ON p.package_id = pb.package_id
		JOIN materials m ON pb.book_id = m.book_id
		WHERE c.class_name = ? AND p.semester = ?
		ORDER BY m.book_id`
	err := db.Select(&details, query, className, semester)
	return details, err
}

// GetMaterialsBySemester 按学期查询所有教材（含关联班级）
type MaterialWithClasses struct {
	Material
	Semester string   `json:"semester"`
	Classes  []string `json:"classes"`
}

func GetMaterialsBySemester(db *sqlx.DB, semester string) ([]MaterialWithClasses, error) {
	if semester == "" {
		return nil, errors.New("学期不能为空")
	}

	// 查询该学期下的所有教材（去重）
	rows := make([]Material, 0)
	err := db.Select(&rows, `SELECT DISTINCT m.book_id, m.isbn, m.title, m.author, m.publisher, m.price, m.created_at, m.extra_info
		FROM materials m
		JOIN package_books pb ON m.book_id = pb.book_id
		JOIN book_packages p ON pb.package_id = p.package_id
		WHERE p.semester = ?
		ORDER BY m.book_id DESC`, semester)
	if err != nil {
		return nil, err
	}

	// 为每本书查关联班级
	result := make([]MaterialWithClasses, 0, len(rows))
	for _, m := range rows {
		var classes []string
		err := db.Select(&classes, `SELECT DISTINCT c.class_name
			FROM package_books pb
			JOIN book_packages p ON pb.package_id = p.package_id
			JOIN class_packages cp ON p.package_id = cp.package_id
			JOIN classes c ON cp.class_id = c.class_id
			WHERE pb.book_id = ? AND p.semester = ?
			ORDER BY c.class_name`, m.BookID, semester)
		if err != nil {
			return nil, err
		}
		if classes == nil {
			classes = []string{}
		}
		result = append(result, MaterialWithClasses{
			Material: m,
			Semester: semester,
			Classes:  classes,
		})
	}
	return result, nil
}

// ============ Excel 导入核心逻辑 ============

// MaterialImportRow 解析后的单行导入数据
type MaterialImportRow struct {
	CourseName string
	ISBN       string
	Title      string
	Publisher  string
	Author     string
	Price      float64
	ClassNames []string
	Department string
	ExtraInfo  string
}

// ImportMaterialsFromRows 事务性地导入教材数据
// semester 必填；academicYear 可选
func ImportMaterialsFromRows(db *sqlx.DB, rows []MaterialImportRow, semester, academicYear string) (inserted int, err error) {
	if semester == "" {
		return 0, errors.New("学期不能为空")
	}

	tx, err := db.Beginx()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, row := range rows {
		if row.ISBN == "" || row.Title == "" {
			continue
		}

		// 1. 查或建教材
		var bookID int
		var existingBook Material
		lookupErr := tx.Get(&existingBook,
			`SELECT book_id, isbn, title, author, publisher, price, created_at, extra_info FROM materials WHERE isbn = ?`,
			row.ISBN)
		if lookupErr == nil {
			bookID = existingBook.BookID
			// 更新可空字段（仅当数据库中对应字段为空时才补全）
			updates := []string{}
			args := []interface{}{}
			if row.Author != "" && !existingBook.Author.Valid {
				updates = append(updates, "author = ?")
				args = append(args, row.Author)
			}
			if row.Publisher != "" && !existingBook.Publisher.Valid {
				updates = append(updates, "publisher = ?")
				args = append(args, row.Publisher)
			}
			if row.Price > 0 && !existingBook.Price.Valid {
				updates = append(updates, "price = ?")
				args = append(args, row.Price)
			}
			if row.ExtraInfo != "" && !existingBook.ExtraInfo.Valid {
				updates = append(updates, "extra_info = ?")
				args = append(args, row.ExtraInfo)
			}
			if len(updates) > 0 {
				args = append(args, bookID)
				updateSQL := "UPDATE materials SET " + strings.Join(updates, ", ") + " WHERE book_id = ?"
				if _, e := tx.Exec(updateSQL, args...); e != nil {
					err = fmt.Errorf("更新教材失败: %w", e)
					return
				}
			}
		} else if errors.Is(lookupErr, sql.ErrNoRows) {
			res, e := tx.Exec(`INSERT INTO materials (isbn, title, author, publisher, price, extra_info)
				VALUES (?, ?, ?, ?, ?, ?)`,
				row.ISBN, row.Title,
				sql.NullString{String: row.Author, Valid: row.Author != ""},
				sql.NullString{String: row.Publisher, Valid: row.Publisher != ""},
				sql.NullFloat64{Float64: row.Price, Valid: row.Price > 0},
				sql.NullString{String: row.ExtraInfo, Valid: row.ExtraInfo != ""},
			)
			if e != nil {
				err = fmt.Errorf("插入教材失败(isbn=%s): %w", row.ISBN, e)
				return
			}
			id, _ := res.LastInsertId()
			bookID = int(id)
		} else {
			err = lookupErr
			return
		}

		// 2. 处理每个班级
		for _, className := range row.ClassNames {
			className = strings.TrimSpace(className)
			if className == "" {
				continue
			}

			grade, major := ParseClassName(className)

			// 查或建班级
			classID, classErr := getOrCreateClassTx(tx, className, grade, major, row.Department)
			if classErr != nil {
				err = classErr
				return
			}

			// 查或建教材包
			packageID, pkgErr := getOrCreatePackageTx(tx, grade, major, semester, academicYear)
			if pkgErr != nil {
				err = pkgErr
				return
			}

			// 教材包-书关联
			if _, e := tx.Exec(`INSERT IGNORE INTO package_books (package_id, book_id, quantity, is_required)
				VALUES (?, ?, 1, 1)`, packageID, bookID); e != nil {
				err = fmt.Errorf("关联教材包失败: %w", e)
				return
			}

			// 班级-教材包关联
			if _, e := tx.Exec(`INSERT IGNORE INTO class_packages (class_id, package_id, academic_year)
				VALUES (?, ?, ?)`, classID, packageID, sql.NullString{String: academicYear, Valid: academicYear != ""}); e != nil {
				err = fmt.Errorf("关联班级失败: %w", e)
				return
			}
		}

		inserted++
	}

	if e := tx.Commit(); e != nil {
		err = e
		return
	}
	return inserted, nil
}

// 在事务中查找或创建班级
func getOrCreateClassTx(tx *sqlx.Tx, className string, grade int, major, department string) (int, error) {
	var classID int
	err := tx.Get(&classID, "SELECT class_id FROM classes WHERE class_name = ?", className)
	if err == nil {
		// 若 department 为空则补全
		if department != "" {
			var curDept sql.NullString
			_ = tx.Get(&curDept, "SELECT department FROM classes WHERE class_id = ?", classID)
			if !curDept.Valid {
				tx.Exec("UPDATE classes SET department = ? WHERE class_id = ?", department, classID)
			}
		}
		return classID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, e := tx.Exec(`INSERT INTO classes (class_name, grade, major, department, student_count)
		VALUES (?, ?, ?, ?, 0)`,
		className,
		sql.NullInt64{Int64: int64(grade), Valid: grade > 0},
		sql.NullString{String: major, Valid: major != ""},
		sql.NullString{String: department, Valid: department != ""},
	)
	if e != nil {
		return 0, e
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// 在事务中查找或创建教材包
func getOrCreatePackageTx(tx *sqlx.Tx, grade int, major, semester, academicYear string) (int, error) {
	query := `SELECT package_id FROM book_packages WHERE semester = ?`
	args := []interface{}{semester}
	if grade > 0 {
		query += " AND grade = ?"
		args = append(args, grade)
	} else {
		query += " AND grade IS NULL"
	}
	if major != "" {
		query += " AND major = ?"
		args = append(args, major)
	} else {
		query += " AND major IS NULL"
	}
	query += " LIMIT 1"

	var packageID int
	err := tx.Get(&packageID, query, args...)
	if err == nil {
		return packageID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	pkgName := buildPackageName(grade, major, semester, academicYear)
	res, e := tx.Exec(`INSERT INTO book_packages (package_name, grade, major, semester, description)
		VALUES (?, ?, ?, ?, ?)`,
		pkgName,
		sql.NullInt64{Int64: int64(grade), Valid: grade > 0},
		sql.NullString{String: major, Valid: major != ""},
		sql.NullString{String: semester, Valid: semester != ""},
		sql.NullString{String: academicYear, Valid: academicYear != ""},
	)
	if e != nil {
		return 0, e
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}
