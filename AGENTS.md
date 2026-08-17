# AGENTS.md — 项目说明书

> 本文件面向 AI 代理与协作者，用于快速理解本仓库的定位、结构、运行方式与约定。
> 核心结论可参考根目录 `README.md`（面向人类的功能介绍），本文件更偏工程视角且与代码保持一致。

## 目录

1. [项目概述](#项目概述)
2. [技术栈](#技术栈)
3. [目录结构](#目录结构)
4. [架构与核心概念](#架构与核心概念)
5. [快速开始](#快速开始)
6. [环境变量](#环境变量)
7. [API 路由概览](#api-路由概览)
8. [数据库表结构](#数据库表结构)
9. [开发指南与常见任务](#开发指南与常见任务)
10. [约定与注意事项](#约定与注意事项)
11. [待补充](#待补充)

---

## 项目概述

一个基于 Go 的**管理后台 API 后端**，同时提供**学生端 API**。为校园综合服务（书籍交易、餐厅/食物、事务、社团、教材、聊天等）提供 RESTful 接口。

- **类型**：Web 后端（纯 API，无前端页面）
- **语言**：Go（模块名 `web-backend`，`go 1.26.2`）
- **分层**：`handlers` → `models` / `services` → `database`，`dto` 提供统一请求/响应格式
- **对外接口分两套**：
  - `/api/*`：管理端（管理员 JWT 认证）
  - `/api/v1/*`：学生端（学生 JWT 认证）

## 技术栈

| 类别 | 技术 | 版本 | 说明 |
|------|------|------|------|
| 语言 | Go | 1.26.2 | `go.mod` 声明 |
| Web 框架 | Gin | v1.12.0 | HTTP 路由、中间件 |
| 数据库 | MySQL | — | 连接串 `charset=utf8mb4` |
| SQL 库 | sqlx | v1.4.0 | 原生 SQL + 结构体映射，非 ORM |
| 驱动 | go-sql-driver/mysql | v1.10.0 | MySQL driver |
| 认证 | golang-jwt/jwt/v5 | v5.2.1 | JWT 签发/解析 |
| 密码 | golang.org/x/crypto | v0.48.0 | bcrypt 哈希 |
| CORS | gin-contrib/cors | v1.7.7 | 跨域白名单 |
| 配置 | godotenv | v1.5.1 | `.env` 加载 |
| Excel | excelize/v2 | v2.10.1 | 表格解析/导入 |
| UUID | google/uuid | v1.6.0 | 唯一标识 |

无测试框架、无 linter、无 Makefile、无 Dockerfile、无 CI 配置。

## 目录结构

```
backend/
├── cmd/
│   ├── server/main.go      # HTTP 服务入口：配置加载、DB 连接、路由注册、静态资源、后台任务
│   └── migrate/main.go     # 独立迁移工具：建库、建表、兼容旧表加列、种子数据
├── internal/               # 内部包（不可被模块外导入）
│   ├── config/config.go    # 环境变量加载与 Config 结构体
│   ├── database/mysql.go   # MySQL 连接池（全局 database.DB）
│   ├── dto/
│   │   ├── request.go      # 请求结构体（带 binding/validate 标签）
│   │   └── response.go     # 统一响应结构 Response + 辅助函数
│   ├── handlers/           # HTTP 处理器（按领域拆分，v1_ 前缀为学生端）
│   ├── middleware/         # auth.go / student_auth.go / v1_auth.go / cors.go / logger.go
│   ├── models/             # 数据模型 + 数据访问函数（*sqlx.DB 为第一参数）
│   └── services/           # 业务逻辑：JWT、Excel 解析、教务验证、云函数客户端
├── .env.example            # 环境变量示例
├── README.md               # 面向人类的功能说明
├── CLAUDE.md               # Claude Code 的简要指引
├── deploy.sh               # 生产部署脚本（交叉编译 + rsync + systemd 重启）
├── local_run.sh            # 本地编译运行脚本（dev/prod）
├── go.mod / go.sum         # 依赖声明
├── LICENSE
└── backend                 # 已编译产物（建议确认是否应加入 .gitignore）
```

### 各层职责与调用链

```
HTTP 请求 → gin 路由 → middleware（Logger → CORS → Auth/StudentAuth/RequireSuperAdmin）
         → handlers（闭包工厂，如 Login(cfg) 返回 gin.HandlerFunc）
         → models（原生 SQL，*sqlx.DB 为第一参数） / services（JWT、Excel、教务、云函数）
         → database.DB（全局 *sqlx.DB 连接池）
```

- **handlers**：每个领域一个文件，均返回 `gin.HandlerFunc`；通过闭包捕获 `*config.Config` 等依赖。统一用 `dto` 包返回。
- **models**：结构体定义 + 数据函数。**不使用 ORM**，直接用参数化 SQL。所有数据函数首参为 `*sqlx.DB`。
- **services**：`auth_service.go`（JWT 生成/解析）、`excel_service.go`（Excel 解析）、`edu_service.go`（教务系统登录验证 + 滑块验证码）、`cloudbase.go`（腾讯 CloudBase 云函数 HTTP 客户端）。

## 架构与核心概念

### 认证与权限（两类主体）

| 主体 | 中间件 | 凭证来源 | 说明 |
|------|--------|----------|------|
| 管理员 | `middleware.Auth` + `RequireSuperAdmin` | `admins` 表 | JWT Bearer；`is_super=1` 才可访问超管路由 |
| 学生 | `middleware.StudentAuth` | `users` 表 | JWT Bearer；登录后校验 `is_frozen` |

- `Auth` 将 `user_id` / `account` / `is_super` 写入 gin context；`StudentAuth` 写入 `student_user_id`。
- 辅助函数：`middleware.GetCurrentAccount / GetCurrentUserID / GetCurrentIsSuper / GetStudentUserID`。

### JWT 与单会话登录

- **令牌永不过期**：`services.GenerateToken` 不设置 `ExpiresAt`，只在用户主动登出时失效。`JWT_EXPIRE_HOURS` 已废弃（`config.go` 仍读取但 `GenerateToken` 忽略）。
- **单会话**：管理员登录将 `admins.is_active = 1`，已激活则拒绝重复登录；登出置 0。
- **启动重置**：`cmd/server` 启动时执行 `resetLoginStatus()`，重置所有管理员会话。
- **会话清理**：后台 goroutine 每 30s 调用 `models.CleanStaleAdminSessions`，清理 5 分钟无活动（`last_active_at`）的会话；心跳端点 `/api/auth/heartbeat` 与 `/api/v1/auth/heartbeat` 用于刷新活跃时间。

### 统一响应格式

所有接口通过 `dto` 返回，结构为 `dto.Response`：

```json
{ "success": true, "data": {}, "message": "操作成功", "total": 100 }
```

辅助函数：`dto.Success` / `SuccessWithTotal` / `SuccessMessage` / `Error` / `BadRequest` / `Unauthorized` / `Forbidden` / `InternalError`。

### 学生注册与教务验证

- `POST /api/v1/auth/register`（幂等）：已注册则更新密码/昵称，未注册则创建用户。
- 密码处理：前端传 RSA 密文 → 后端先 SHA-256 再 bcrypt。
- 教务验证：`services.VerifySchoolCredentials`（当前仅 `hbut`，见 `edu_service.go`），内部通过超星滑块 + CloudBase 云函数自动求解验证码；`SKIP_EDU_VERIFY=true` 可跳过。

## 快速开始

```bash
# 1. 配置环境变量
cp .env.example .env

# 2. 数据库迁移（自动建库、建表、加兼容列、写入种子数据，默认管理员 admin/admin123）
go run ./cmd/migrate

# 3. 启动服务（默认 :3001）
go run ./cmd/server

# 4. 编译二进制
go build -o server ./cmd/server
```

### 脚本

| 命令 | 说明 |
|------|------|
| `./local_run.sh [dev|prod]` | 编译到 `./dist/backend-local` 并前台运行；默认加载 `.env.dev`，`prod` 加载 `.env.prod` |
| `./deploy.sh` | 交叉编译（`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`）→ rsync 到 SSH 别名 `server` 的 `/home/www/project/go` → 重启 systemd 服务 `go-app`；读取本地 `.env.prod` |

部署前需配置 `.env.prod`、SSH 别名 `server`，并在服务器上准备 `go-app` systemd 服务。

## 环境变量

以 `.env.example` 为准；`config.go` 提供兜底默认值（与示例略有差异，见下）。

| 变量 | 示例/默认 | 必填 | 用途 |
|------|-----------|------|------|
| `PORT` | `3001` | 否 | 服务监听端口 |
| `GIN_MODE` | `debug` | 否 | Gin 运行模式（debug/release） |
| `DB_HOST` | `127.0.0.1` | 是 | MySQL 主机 |
| `DB_PORT` | `3306` | 是 | MySQL 端口 |
| `DB_USER` | `root` | 是 | MySQL 用户名 |
| `DB_PASSWORD` | 空 | 是 | MySQL 密码 |
| `DB_NAME` | `admin_panel` | 是 | 数据库名 |
| `JWT_SECRET` | 占位符 | **生产必改** | JWT 签名密钥 |
| `JWT_EXPIRE_HOURS` | `24` | 否 | **已废弃**（令牌永不过期，仅读取不生效） |
| `FRONTEND_URL` | `http://localhost:5173` | 否 | CORS 白名单（单值，`FRONTEND_URLS` 的兜底） |
| `FRONTEND_URLS` | 逗号分隔 | 否 | CORS 白名单（多值，优先于 `FRONTEND_URL`） |
| `UPLOAD_DIR` | `./uploads` | 否 | 上传文件目录（自动创建） |
| `BASE_URL` | 空 | 否 | 用于把相对图片路径转绝对 URL（`handlers.SetBaseURL`） |
| `SKIP_EDU_VERIFY` | `false` | 否 | 跳过教务系统验证（生产无法直连教育网时设 `true`） |
| `CAPTCHA_SERVICE_URL` | `http://127.0.0.1:9999` | 否 | Python 滑块求解微服务地址 |
| `CLOUDBASE_ENV_ID` | 未在示例中 | 条件 | CloudBase 环境 ID（`cloudbase.go` 直读） |
| `CLOUDBASE_ACCESS_TOKEN` | 未在示例中 | 条件 | CloudBase 访问令牌（`cloudbase.go` 直读） |

> 注意：`config.go` 中 `DB_USER/DB_PASSWORD/DB_NAME` 的兜底值分别为 `name/password/dbname`，与 `.env.example` 不一致；实际以 `.env` 为准。

## API 路由概览

路由定义集中在 `cmd/server/main.go`。除以下公开接口外，其余均需 JWT：

- 公开（无认证）：`GET /api/health`、`POST /api/captcha/solve`、`POST /api/auth/login`、`POST /api/v1/auth/register`、`GET /api/v1/auth/check-user`

### 管理端 `/api`（`Auth` 中间件，`RequireSuperAdmin` 标注为超管）

| 模块 | 端点 | 说明 |
|------|------|------|
| 认证 | `POST /auth/logout`、`GET /auth/me`、`PUT /auth/change-password`、`POST /auth/heartbeat` | 登录后可用 |
| 模块 | `GET /modules` | 前端导航模块列表 |
| 管理员（超管） | `GET/POST /admin/admins`、`PUT /admin/admins/:id`、`PUT /admin/admins/:id/info`、`DELETE /admin/admins/:id` | 管理员 CRUD |
| 用户 | `GET/POST /users`、`GET/PUT /users/:id` | 普通用户 |
| 用户（超管） | `DELETE /users/:id`、`DELETE /users/:id/hard`、`PUT /users/:id/freeze` | 软删/硬删/冻结 |
| 书籍分类 | `GET/POST /book/categories`、`GET /book/categories/detail`、`PUT/DELETE /book/categories/:id` | 支持 `/books/` 复数 |
| 书籍 | `GET/POST /book`、`GET/PUT/DELETE /book/:id`、`POST /book/upload-image` | 支持 `/books/` 复数 |
| 餐厅 | `GET/POST/PUT/DELETE /shops` | `PUT/DELETE` 为 `/shops/:id` |
| 食物 | `GET/POST/PUT/DELETE /foods` | 同上 |
| 事务 | `GET/POST/PUT/DELETE /affairs` | 同上 |
| 事务种类 | `GET/POST/PUT/DELETE /affair-categories` | 同上 |
| 社团 | `GET/POST /clubs`、`GET /clubs/categories`、`GET/PUT/DELETE /clubs/:id` | 含分类 |
| 会话（管理员） | `GET /conversations/user/:userId`、`GET /conversations/:id/messages`、`DELETE /conversations/:id` | 聊天管理 |
| Excel | `POST /excel/import?table=shops\|foods\|affairs`、`POST /excel/preview` | 通用导入/预览 |
| 教材 | `GET/POST /materials`、`GET /materials/semesters`、`GET /materials/classes`、`PUT/DELETE /materials/:id`、`POST /materials/import`、`POST /materials/preview` | 教材管理 |

### 学生端 `/api/v1`（`StudentAuth` 中间件）

| 模块 | 端点 | 说明 |
|------|------|------|
| 认证 | `POST /auth/register`、`GET /auth/check-user`（公开）、`GET /auth/me`、`POST /auth/heartbeat` | 注册/登录幂等 |
| 社团 | `GET/POST /clubs`、`GET /clubs/categories`、`GET /clubs/:id` | 浏览/申请创建 |
| 书籍 | `GET/POST /book`、`GET /book/mine`、`GET/PUT/DELETE /book/:id`、`POST /book/upload-image`、`POST /book/upload`、`DELETE /book/:id/images/:imageId`、`POST /book/:id/want` | 支持 `/books/` 复数 |
| 食物 | `GET /foods`、`GET /foods/filters` | 列表/筛选条件 |
| 教材 | `GET /materials?semester=&class_name=`、`GET /materials/semesters`、`GET /materials/classes`、`GET /materials/:id` | `semester` 必填 |
| 会话 | `GET/POST /conversations`、`GET/POST /conversations/:id/messages` | 聊天 |

- 静态资源：`/uploads` → `cfg.UploadDir`。
- 请求路径会做斜杠归一化（折叠重复 `/`，避免反向代理拼出的 `//` 导致 404）。

## 数据库表结构

所有表在 `cmd/migrate/main.go` 中定义（`CREATE TABLE IF NOT EXISTS` + 针对旧表的 `ALTER TABLE ADD COLUMN` 兼容逻辑），字符集 `utf8mb4`。

- **账号**：`admins`（管理员，含 `is_super`、`is_active`、`last_active_at`、`schoolId`）、`users`（学生，含 `stuId`、`nickName`、`schoolId`、`password_hash`、`isDeleted`、`is_frozen`、`last_active_at`）
- **书籍**：`book`（含多列兼容字段）、`book_images`、`book_wants`、`book_categories`
- **餐厅/食物**：`shops`、`foods`（含 `school_id`、`canteen_name` 等）
- **事务**：`affairs`、`affair_categories`
- **社团**：`clubs`（外键 `principal_id` → `users.id`，`ON DELETE SET NULL`）
- **聊天**：`conversation`、`message`
- **交易**：`purchase`
- **教材**：`materials`、`classes`、`book_packages`、`package_books`、`class_packages`

种子数据：`book_categories`（8 个默认分类）、默认管理员 `admin/admin123`（`is_super=1`）。

## 开发指南与常见任务

### 新增一个业务模块（示例：新增 `X` 领域）

1. `internal/models/x.go`：定义结构体（字段需加 `db` tag）+ 数据访问函数（首参 `*sqlx.DB`，参数化 SQL）。
2. `internal/handlers/x.go`（学生端用 `v1_x.go`）：定义 `gin.HandlerFunc`（闭包工厂，需要依赖时捕获 `cfg`）。
3. `internal/dto/request.go`：新增请求结构体。
4. `cmd/server/main.go`：注册路由到对应分组（管理端 `authorized` / 学生端 `v1Auth`）。
5. 若需建表：在 `cmd/migrate/main.go` 追加 `CREATE TABLE IF NOT EXISTS`。

### 修改数据库结构

- 直接编辑 `cmd/migrate/main.go`，新增表或对旧表用 `ALTER TABLE ADD COLUMN` 做兼容（参考现有「先查 `INFORMATION_SCHEMA.COLUMNS` 再 ALTER」的模式）。
- 执行 `go run ./cmd/migrate` 应用变更。

### 调试

- 运行 `go run ./cmd/server` 查看请求日志（`middleware.Logger` 输出 `[状态码] 方法 路径 耗时`）。
- 健康检查：`GET /api/health`（内部校验 DB ping）。
- 关闭教务验证以本地联调：`.env` 中设 `SKIP_EDU_VERIFY=true`。
- 无单元测试；如需验证可临时用 `curl` 调接口（先 `POST /api/auth/login` 拿 token）。

## 约定与注意事项

- **响应**：一律用 `dto` 包辅助函数，禁止直接 `c.JSON` 拼返回。
- **SQL**：参数化查询，禁止字符串拼接用户输入。
- **`db` tag**：所有映射到数据库列的 struct 字段必须显式写 `db:"..."`，否则 sqlx 映射失败（历史坑，见 `models/material.go`）。
- **可空类型**：DB 可空列用 `sql.NullString` / `sql.NullFloat64` / `sql.NullTime`，对外响应前展平成 `string` / `float64` / 格式化时间。
- **时间格式**：API 返回时间统一 `2006-01-02T15:04:05`（`models.formatTime`）。
- **教材查询**：学生端 `semester` 参数必填；`classes` 字段不使用 `omitempty`，保证空数组也返回（避免前端缺失字段）。
- **路径兼容**：书籍相关接口同时注册单数 `/book/...` 与复数 `/books/...`。
- **命名**：文件小写蛇形（`affair_category.go`），结构体大驼峰（`CreateAdminRequest`）。
- **注释**：代码中注释以中文为主，与现有风格保持一致。
- **敏感信息**：`internal/services/edu_service.go` 硬编码了教务系统地址（`jwxt.hbut.edu.cn`）与验证码 `captchaId`；CloudBase 凭据走环境变量。文档与提交中请勿泄露真实密钥。

## 待补充

- 无测试、无 linter、无 Makefile、无 CI、无 Dockerfile；如需工程化可后续补齐。
- `docs/swagger` 目录为空：handler 中有 swagger 注释但未生成文档。
- 根目录存在 `backend` 编译产物文件，建议确认是否应加入 `.gitignore`（当前 `.gitignore` 仅忽略 `/server`、`/migrate`、`/dist`）。
- `middleware.V1Auth`（基于 body 的 `id/stuId/schoolId` 认证）目前未被路由使用，路由统一走 `StudentAuth`（JWT），该中间件可能为遗留代码。
