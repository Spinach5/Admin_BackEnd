# Web Backend

## 项目概述

这是一个基于 Go 语言开发的管理后台 API 后端服务，采用 Gin 框架构建，为管理后台系统提供完整的 RESTful API 接口支持。

系统主要功能包括：
- **用户认证与权限管理**：支持管理员登录、JWT 令牌认证、超级管理员权限控制
- **学生端接口**：面向学生的独立 API 端点，支持书籍交易、社团管理、教材查询等
- **多模块管理**：涵盖书籍、餐厅、食物、事务、社团、教材等多个业务模块
- **Excel 数据导入**：支持 Excel 文件的预览和批量导入功能
- **聊天系统**：为书籍交易提供买卖双方的即时沟通功能
- **文件上传**：支持书籍图片等资源的上传与管理

## 技术栈

| 类别 | 技术 | 版本 | 说明 |
|------|------|------|------|
| **语言** | Go | 1.26.2 | 主要开发语言 |
| **Web 框架** | Gin | v1.12.0 | 轻量级 HTTP 框架 |
| **数据库** | MySQL | - | 关系型数据库 |
| **数据库驱动** | sqlx | v1.4.0 | SQL 扩展工具库 |
| **ORM 风格** | raw SQL | - | 使用参数化 SQL 查询 |
| **认证** | JWT (jwt/v5) | v5.2.1 | Bearer Token 认证 |
| **密码加密** | bcrypt | v0.48.0 | 密码哈希加密 |
| **CORS** | gin-contrib/cors | v1.7.7 | 跨域资源共享 |
| **配置管理** | godotenv | v1.5.1 | 环境变量加载 |
| **Excel 处理** | excelize/v2 | v2.10.1 | Excel 文件读写 |
| **UUID** | google/uuid | v1.6.0 | 唯一标识符生成 |

## 项目结构

```
backend/
├── cmd/                          # 应用入口
│   ├── server/                   # 服务端入口
│   │   └── main.go              # HTTP 服务器启动、路由定义
│   └── migrate/                  # 数据库迁移工具
│       └── main.go              # 创建数据库、表结构、种子数据
├── internal/                     # 内部包（不可外部导入）
│   ├── config/                   # 配置管理
│   │   └── config.go            # 环境变量加载、配置结构体
│   ├── database/                 # 数据库连接
│   │   ├── mysql.go             # MySQL 连接池初始化
│   ├── dto/                      # 数据传输对象
│   │   ├── request.go           # 请求结构体定义
│   │   └── response.go          # 响应格式封装、辅助函数
│   ├── handlers/                 # HTTP 处理器
│   │   ├── auth.go              # 认证相关（登录、登出、密码修改）
│   │   ├── admin.go             # 管理员管理（CRUD）
│   │   ├── user.go              # 用户管理
│   │   ├── book.go              # 书籍管理
│   │   ├── food.go              # 食物管理
│   │   ├── shop.go              # 餐厅管理
│   │   ├── affair.go            # 事务管理
│   │   ├── affair_category.go   # 事务种类管理
│   │   ├── club.go              # 社团管理
│   │   ├── material.go          # 教材管理
│   │   ├── chat.go              # 聊天功能
│   │   ├── excel.go             # Excel 导入
│   │   ├── captcha.go           # 验证码
│   │   ├── modules.go           # 模块列表
│   │   ├── health.go            # 健康检查
│   │   ├── v1.go                # V1 版本学生路由
│   │   ├── v1_auth.go           # 学生认证
│   │   ├── v1_book.go           # 学生端书籍
│   │   ├── v1_food.go           # 学生端食物
│   │   └── v1_material.go       # 学生端教材
│   ├── middleware/               # 中间件
│   │   ├── auth.go              # 管理员 JWT 认证
│   │   ├── student_auth.go      # 学生 JWT 认证
│   │   ├── v1_auth.go           # V1 认证中间件
│   │   ├── cors.go              # CORS 跨域处理
│   │   └── logger.go            # 请求日志记录
│   ├── models/                   # 数据模型
│   │   ├── admin.go             # 管理员数据操作
│   │   ├── user.go              # 用户数据操作
│   │   ├── book.go              # 书籍数据操作
│   │   ├── food.go              # 食物数据操作
│   │   ├── shop.go              # 餐厅数据操作
│   │   ├── affair.go            # 事务数据操作
│   │   ├── affair_category.go   # 事务种类数据操作
│   │   ├── club.go              # 社团数据操作
│   │   ├── material.go          # 教材数据操作
│   │   └── chat.go              # 聊天数据操作
│   ├── services/                 # 业务服务
│   │   ├── auth_service.go      # JWT 令牌生成与解析
│   │   ├── excel_service.go     # Excel 解析服务
│   │   ├── edu_service.go       # 教务系统验证
│   │   └── cloudbase.go         # 云服务集成
├── docs/                         # 文档
│   └── swagger/                  # Swagger 文档（待生成）
├── .env.example                  # 环境变量示例
├── .gitignore                    # Git 忽略规则
├── go.mod                        # Go 模块依赖
├── go.sum                        # 依赖校验
├── deploy.sh                     # 部署脚本
├── local_run.sh                  # 本地运行脚本
└── CLAUDE.md                     # AI 辅助配置
```

### 模块说明

#### `cmd/server`
服务主入口，负责：
- 加载配置、初始化数据库连接
- 创建 Gin 引擎、注册全局中间件
- 定义路由分组与 API 端点
- 启动 HTTP 服务器

#### `cmd/migrate`
独立的数据库迁移工具：
- 自动创建数据库（如不存在）
- 创建所有业务表结构
- 添加兼容列（支持旧表升级）
- 插入种子数据（默认分类、管理员账号等）

#### `internal/handlers`
请求处理器层，每个文件对应一个业务领域：
- 采用闭包工厂模式（`Login(cfg)` 返回 `gin.HandlerFunc`）
- 通过闭包注入依赖，实现关注点分离
- 统一使用 `dto` 包的响应函数返回标准格式

#### `internal/models`
数据访问层：
- 直接使用 `*sqlx.DB` 执行参数化 SQL
- 不使用 ORM，保持透明可控的数据库操作
- 封装增删改查的核心业务查询

#### `internal/middleware`
HTTP 中间件层：
- **Auth**：管理员 JWT Bearer Token 验证
- **StudentAuth**：学生端 JWT 验证
- **RequireSuperAdmin**：超级管理员权限检查
- **CORS**：跨域资源共享处理
- **Logger**：请求日志记录

#### `internal/services`
业务逻辑服务：
- 认证服务（JWT 生成与解析、密码加密）
- Excel 解析与导入服务
- 教务系统对接服务

## 构建方法

### 环境要求

- Go 1.26+
- MySQL 5.7+ 或 MySQL 8.0+
- Git（用于版本管理）

### 快速开始

#### 1. 克隆项目

```bash
git clone <repository-url>
cd backend
```

#### 2. 配置环境变量

复制示例配置文件并根据实际情况修改：

```bash
cp .env.example .env
```

`.env` 文件配置说明：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `3001` | 服务端口 |
| `GIN_MODE` | `debug` | Gin 运行模式（debug/release） |
| `DB_HOST` | `127.0.0.1` | MySQL 主机地址 |
| `DB_PORT` | `3306` | MySQL 端口 |
| `DB_USER` | `root` | 数据库用户名 |
| `DB_PASSWORD` | 空 | 数据库密码 |
| `DB_NAME` | `admin_panel` | 数据库名 |
| `JWT_SECRET` | - | JWT 签名密钥（生产环境必须修改） |
| `FRONTEND_URL` | `http://localhost:5173` | 前端地址（CORS 白名单） |
| `CAPTCHA_SERVICE_URL` | `http://127.0.0.1:9999` | 验证码服务地址 |
| `SKIP_EDU_VERIFY` | `false` | 是否跳过教务系统验证 |

#### 3. 初始化数据库

执行数据库迁移，自动创建数据库、表结构和默认数据：

```bash
go run ./cmd/migrate
```

迁移完成后会自动创建：
- 默认管理员账号：`admin` / `admin123`
- 书籍分类种子数据
- 数据库表结构

#### 4. 启动服务

开发模式运行：

```bash
go run ./cmd/server
```

或者编译为二进制后运行：

```bash
# 编译
go build -o server ./cmd/server

# 运行
./server
```

服务启动后访问 `http://localhost:3001` 即可。

### API 路由概览

#### 管理员 API (`/api`)

| 路径 | 方法 | 描述 | 认证 |
|------|------|------|------|
| `/api/health` | GET | 健康检查 | 否 |
| `/api/auth/login` | POST | 管理员登录 | 否 |
| `/api/auth/logout` | POST | 管理员登出 | 是 |
| `/api/auth/me` | GET | 获取当前用户信息 | 是 |
| `/api/auth/change-password` | PUT | 修改密码 | 是 |
| `/api/admin/admins` | GET/POST | 管理员列表/创建 | 是（超级管理员） |
| `/api/admin/admins/:id` | PUT/DELETE | 更新/删除管理员 | 是（超级管理员） |
| `/api/users` | GET/POST | 用户列表/创建 | 是 |
| `/api/users/:id` | GET/PUT/DELETE | 用户详情/更新/删除 | 是 |
| `/api/book` | GET/POST | 书籍列表/创建 | 是 |
| `/api/book/:id` | GET/PUT/DELETE | 书籍详情/更新/删除 | 是 |
| `/api/book/categories` | GET/POST | 书籍分类管理 | 是 |
| `/api/shops` | GET/POST | 餐厅列表/创建 | 是 |
| `/api/foods` | GET/POST | 食物列表/创建 | 是 |
| `/api/affairs` | GET/POST | 事务列表/创建 | 是 |
| `/api/clubs` | GET/POST | 社团列表/创建 | 是 |
| `/api/materials` | GET/POST | 教材列表/创建 | 是 |
| `/api/excel/preview` | POST | Excel 预览 | 是 |
| `/api/excel/import` | POST | Excel 导入 | 是 |

#### 学生端 API (`/api/v1`)

| 路径 | 方法 | 描述 | 认证 |
|------|------|------|------|
| `/api/v1/auth/register` | POST | 学生注册 | 否 |
| `/api/v1/auth/check-user` | GET | 检查用户是否存在 | 否 |
| `/api/v1/auth/me` | GET | 获取当前学生信息 | 是 |
| `/api/v1/book` | GET/POST | 书籍列表/发布 | 是 |
| `/api/v1/book/mine` | GET | 我的书籍 | 是 |
| `/api/v1/book/:id` | GET/PUT/DELETE | 书籍详情/更新/删除 | 是 |
| `/api/v1/book/:id/want` | POST | 想要此书 | 是 |
| `/api/v1/foods` | GET | 食物列表 | 是 |
| `/api/v1/foods/filters` | GET | 食物筛选条件 | 是 |
| `/api/v1/materials` | GET | 教材列表 | 是 |
| `/api/v1/clubs` | GET/POST | 社团列表/申请创建 | 是 |
| `/api/v1/conversations` | GET/POST | 会话列表/创建 | 是 |
| `/api/v1/conversations/:id/messages` | GET/POST | 消息列表/发送 | 是 |

### API 响应格式

所有 API 响应均使用统一的 JSON 格式：

```json
{
  "success": true,
  "data": { ... },
  "message": "操作成功",
  "total": 100
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | boolean | 请求是否成功 |
| `data` | any | 响应数据 |
| `message` | string | 提示消息（可选） |
| `total` | number | 数据总数（可选，分页场景） |

错误响应示例：

```json
{
  "success": false,
  "message": "未提供认证令牌"
}
```

### 部署

使用部署脚本：

```bash
# 赋予执行权限
chmod +x deploy.sh

# 执行部署
./deploy.sh
```

或使用本地运行脚本：

```bash
chmod +x local_run.sh
./local_run.sh
```

## 数据库表结构

### 管理员与用户

- **admins**：管理员表（account、password、is_super、is_active）
- **users**：普通用户表（stuId、nickName、schoolId、password_hash）

### 业务表

- **book / book_images / book_wants**：书籍交易相关
- **book_categories**：书籍分类
- **shops / foods**：餐厅与食物
- **affairs / affair_categories**：事务与分类
- **clubs**：社团信息
- **conversation / message**：聊天会话与消息
- **purchase**：购买记录

### 教材管理

- **materials**：教材信息
- **classes**：班级信息
- **book_packages / package_books**：教材包及其明细
- **class_packages**：班级与教材包关联

## 注意事项

1. **JWT 令牌**：令牌永不过期，仅在用户主动注销时失效。后台定时任务会在 5 分钟无活动后自动清理活跃状态。
2. **单会话登录**：同一账号不允许同时多点登录，新登录会使旧会话失效。
3. **数据库字符集**：所有表使用 `utf8mb4` 字符集，支持中文和 emoji。
4. **路径兼容**：书籍相关 API 同时支持单数形式 (`/book/...`) 和复数形式 (`/books/...`)。
5. **文件上传**：上传文件存储在 `./uploads` 目录，通过 `/uploads/*` 路径访问。
6. **时间格式**：API 返回的时间统一使用 `2006-01-02T15:04:05` 格式。

## 开发规范

- Go 代码不添加注释（遵循项目约定）
- 使用 `dto` 包的响应函数统一返回格式
- 数据库操作使用参数化 SQL，禁止字符串拼接
- 新功能应放置在对应的 handler、model、service 包中
- 遵循现有命名规范：小写蛇形文件命名，大驼峰结构体命名
