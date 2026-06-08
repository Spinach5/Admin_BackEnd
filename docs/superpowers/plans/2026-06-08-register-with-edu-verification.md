# 注册接口教务系统验证 & 用户检查接口 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在注册接口中接入 HBUT 教务系统验证，确保只有真实学生可注册；新增公开的用户存在性检查接口。

**Architecture:** 新增 `edu_service.go` 封装 RSA 加密 + HTTP 模拟登录；在现有 `StudentRegister` handler 中插入学校白名单校验和教务验证步骤；新增 `CheckUser` handler 查询用户存在性。

**Tech Stack:** Go 1.x, gin, sqlx, MySQL, crypto/rsa, crypto/x509, net/http

---

### Task 1: 创建教务验证服务 `internal/services/edu_service.go`

**Files:**
- Create: `internal/services/edu_service.go`

- [ ] **Step 1: 创建 edu_service.go**

```go
package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// hbutPublicKey 来自 auth.js，与前端 JSEncrypt 使用的公钥一致
const hbutPublicKey = "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDcwU0RBrR31L3eHKVGogsJKdr36D3rrjUNaZ77yxxO9HSIojA4jyJylCVALkcu4cK+bbGLpedilJSlcyohso+IBI+A/eAfjS/GhIT/OWEsg8/+YLt+asM8+pdISE/T14tTqg/WDe8nqX48dazB0Izu1ytaPPFRWuYqtUTRpZ7IsQIDAQAB"

// VerifyHbutCredentials 模拟登录 HBUT 教务系统验证账号密码
// 成功返回 nil，失败返回 error
func VerifyHbutCredentials(stuID, password string) error {
	// 1. RSA 加密密码
	encrypted, err := rsaEncrypt(password, hbutPublicKey)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 2. 构造 form 请求体
	form := url.Values{}
	form.Set("username", stuID)
	form.Set("password", encrypted)
	form.Set("rememberMe", "1")

	// 3. 发送 POST 请求
	req, err := http.NewRequest("POST", "https://jwxt.hbut.edu.cn/admin/login",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Referer", "https://jwxt.hbut.edu.cn")
	req.Header.Set("Origin", "https://jwxt.hbut.edu.cn")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("教务系统验证失败，请稍后重试: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	bodyStr := string(body)

	// 4. 判断结果：返回 JSON 表示失败
	if strings.HasPrefix(bodyStr, "{") || strings.HasPrefix(bodyStr, "[") {
		var result struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Msg     string `json:"msg"`
		}
		if jsonErr := json.Unmarshal(body, &result); jsonErr == nil {
			if result.Code != 0 && result.Code != 200 {
				msg := result.Message
				if msg == "" {
					msg = result.Msg
				}
				if msg == "" {
					msg = "学号或密码错误"
				}
				return errors.New(msg)
			}
		}
	}

	// HTML 响应 + HTTP 200 = 登录成功
	return nil
}

// rsaEncrypt 使用 RSA PKCS1v15 加密明文，返回 Base64 密文
func rsaEncrypt(plaintext, pubKeyBase64 string) (string, error) {
	derBytes, err := base64.StdEncoding.DecodeString(pubKeyBase64)
	if err != nil {
		return "", fmt.Errorf("公钥解码失败: %w", err)
	}

	pub, err := x509.ParsePKIXPublicKey(derBytes)
	if err != nil {
		return "", fmt.Errorf("公钥解析失败: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("不是有效的 RSA 公钥")
	}

	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, []byte(plaintext))
	if err != nil {
		return "", fmt.Errorf("RSA 加密失败: %w", err)
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./internal/services/
```

预期：编译通过，无错误。

- [ ] **Step 3: 提交**

```bash
git add internal/services/edu_service.go
git commit -m "feat: 添加 HBUT 教务系统凭证验证服务

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: 修改注册接口，加入学校校验和教务验证

**Files:**
- Modify: `internal/handlers/v1_auth.go`

- [ ] **Step 1: 在 StudentRegister 中插入学校白名单校验和教务验证逻辑**

将现有 `StudentRegister` 函数体替换为以下版本（在学校校验和查重之间插入教务验证）：

找到函数中的这段代码：

```go
	if len(req.Password) < 6 {
		dto.BadRequest(c, "密码至少6位")
		return
	}

	_, err := models.GetUserByStuIDAndSchoolID(database.DB, req.StuID, req.SchoolID)
```

在上面的代码块之后、查重之前，插入以下两段：

```go
	// 学校白名单校验
	if req.SchoolID != "hbut" {
		dto.BadRequest(c, "暂不支持该学校")
		return
	}

	// 教务系统凭证验证
	if err := services.VerifyHbutCredentials(req.StuID, req.Password); err != nil {
		log.Printf("教务系统验证失败 stuId=%s: %v", req.StuID, err)
		dto.BadRequest(c, "学号或密码错误")
		return
	}
```

**说明：** 插入位置在密码长度校验之后、用户查重之前。完整函数变为：

```go
func StudentRegister(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.StudentRegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "请填写所有必填字段")
			return
		}

		if len(req.Password) < 6 {
			dto.BadRequest(c, "密码至少6位")
			return
		}

		// 新增：学校白名单校验
		if req.SchoolID != "hbut" {
			dto.BadRequest(c, "暂不支持该学校")
			return
		}

		// 新增：教务系统凭证验证
		if err := services.VerifyHbutCredentials(req.StuID, req.Password); err != nil {
			log.Printf("教务系统验证失败 stuId=%s: %v", req.StuID, err)
			dto.BadRequest(c, "学号或密码错误")
			return
		}

		_, err := models.GetUserByStuIDAndSchoolID(database.DB, req.StuID, req.SchoolID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("查询用户失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}
		if err == nil {
			dto.BadRequest(c, "该学号在此学校已注册")
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			log.Printf("密码加密失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		user := &models.User{
			StuID:        req.StuID,
			NickName:     req.NickName,
			SchoolID:     req.SchoolID,
			PasswordHash: string(hashed),
		}

		if err := models.CreateUserWithPassword(database.DB, user); err != nil {
			log.Printf("创建用户失败: %v", err)
			dto.InternalError(c, "注册失败")
			return
		}

		expireHours := parseExpireHours(cfg)
		token, err := services.GenerateToken(user.ID, "", 0, cfg.JWTSecret, expireHours)
		if err != nil {
			log.Printf("生成token失败: %v", err)
			dto.InternalError(c, "服务器错误")
			return
		}

		dto.Success(c, gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"stuId":    user.StuID,
				"nickName": user.NickName,
				"schoolId": user.SchoolID,
			},
		})
	}
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./internal/handlers/
```

预期：编译通过，无错误。

- [ ] **Step 3: 提交**

```bash
git add internal/handlers/v1_auth.go
git commit -m "feat: 注册接口加入学校白名单和教务系统验证

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: 新增用户存在性检查接口 CheckUser

**Files:**
- Modify: `internal/handlers/v1_auth.go`

- [ ] **Step 1: 在 v1_auth.go 末尾添加 CheckUser handler**

在文件末尾追加以下函数：

```go
// CheckUser 检查用户是否存在 (公开接口)
// @Summary 检查用户是否存在
// @Description 通过学号和学校代码查询用户是否已注册
// @Tags 认证
// @Produce json
// @Param stuId query string true "学号"
// @Param schoolId query string true "学校代码"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/check-user [get]
func CheckUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		stuID := c.Query("stuId")
		schoolID := c.Query("schoolId")

		if stuID == "" || schoolID == "" {
			dto.BadRequest(c, "请提供 stuId 和 schoolId")
			return
		}

		_, err := models.GetUserByStuIDAndSchoolID(database.DB, stuID, schoolID)
		exists := err == nil

		dto.Success(c, gin.H{
			"stuId":    stuID,
			"schoolId": schoolID,
			"exists":   exists,
		})
	}
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./internal/handlers/
```

预期：编译通过，无错误。

- [ ] **Step 3: 提交**

```bash
git add internal/handlers/v1_auth.go
git commit -m "feat: 新增用户存在性检查接口 CheckUser

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: 注册检查接口路由

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: 在 V1 公开路由组添加 check-user 路由**

在 `cmd/server/main.go` 中找到 V1 路由组：

```go
		v1 := r.Group("/api/v1")
		{
			v1.POST("/auth/register", handlers.StudentRegister(cfg))
			v1.POST("/auth/login", handlers.StudentLogin(cfg))
```

在 `v1.POST("/auth/login", handlers.StudentLogin(cfg))` 之后添加：

```go
			v1.GET("/auth/check-user", handlers.CheckUser())
```

完整代码块：

```go
		v1 := r.Group("/api/v1")
		{
			v1.POST("/auth/register", handlers.StudentRegister(cfg))
			v1.POST("/auth/login", handlers.StudentLogin(cfg))
			v1.GET("/auth/check-user", handlers.CheckUser())

			v1Auth := v1.Group("")
```

- [ ] **Step 2: 编译验证**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./cmd/server/
```

预期：编译通过，无错误。

- [ ] **Step 3: 提交**

```bash
git add cmd/server/main.go
git commit -m "feat: 注册 check-user 路由

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: 整体编译与验证

**Files:**
- 无新建或修改（仅验证）

- [ ] **Step 1: 完整项目编译**

```bash
cd /home/zqw/biancheng/Project/backend && go build -o server ./cmd/server
```

预期：编译通过，生成 `server` 二进制。无错误无警告。

- [ ] **Step 2: 检查 go vet**

```bash
cd /home/zqw/biancheng/Project/backend && go vet ./...
```

预期：无报错。

- [ ] **Step 3: 验证路由注册**

```bash
cd /home/zqw/biancheng/Project/backend && grep -n "check-user" cmd/server/main.go
```

预期：输出匹配行，确认路由已注册。

- [ ] **Step 4: 提交（如有变更）**

```bash
git status
```

如果 Task 5 无文件变更，无需额外提交。
