# 注册接口教务系统验证 & 用户检查接口

**日期:** 2026-06-08  
**状态:** 设计中  
**影响范围:** `internal/services/edu_service.go`(新), `internal/handlers/v1_auth.go`, `cmd/server/main.go`

## 背景

当前学生注册接口 (`POST /api/v1/auth/register`) 直接接受学号和密码并创建用户，未验证学生在教务系统的身份真实性。需要加入教务系统登录验证，确保只有真实学生可以注册。

## 需求

### FR1: 修改注册接口 — 教务系统验证
- 接收参数：`schoolId`（学校代码）、`stuId`（学号）、`password`（密码）、`nickName`（昵称）
- **仅处理 `schoolId = "hbut"`**，其他学校返回 `"暂不支持该学校"`
- 模拟登录 HBUT 教务系统 (`https://jwxt.hbut.edu.cn/admin/login`)，验证账号密码
- 教务系统验证通过 → 保存用户到数据库（bcrypt 加密密码），返回 JWT token
- 教务系统验证失败 → 返回错误，不创建用户
- 教务系统网络异常 → 返回错误，不创建用户

### FR2: 新增用户存在性检查接口
- 传入 `stuId` + `schoolId`
- 返回该用户是否已注册
- **公开接口**，无需登录

## 设计

### 文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/services/edu_service.go` | **新增** | HBUT 教务系统登录模拟 |
| `internal/handlers/v1_auth.go` | **修改** | StudentRegister 插入教务验证；新增 CheckUser handler |
| `cmd/server/main.go` | **修改** | 新增 GET `/api/v1/auth/check-user` 路由 |

### 1. 教务验证服务 `internal/services/edu_service.go`

```go
package services

// VerifyHbutCredentials(stuID, password string) error
```

**逻辑（参考 auth.js）：**

1. **RSA 加密密码**: 使用与 JS 相同的公钥 `MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDc...`，`crypto/rsa` 加密后 Base64 输出
2. **HTTP 请求**: `POST https://jwxt.hbut.edu.cn/admin/login`
   - Header: `Content-Type: application/x-www-form-urlencoded; charset=UTF-8`
   - Header: `Referer: https://jwxt.hbut.edu.cn`
   - Header: `Origin: https://jwxt.hbut.edu.cn`
   - Body: `username=<stuId>&password=<encrypted>&rememberMe=1`
   - Timeout: 10s
3. **结果判断**:
   - HTTP 200 + 响应体是 HTML（非 JSON）→ 登录成功
   - 响应体是 JSON 且 `code != 0/200` → 返回错误信息
   - 网络错误 / 超时 → 返回错误

### 2. 修改注册接口 `StudentRegister`

在现有流程中插入两个步骤：

```
参数校验 → 学校校验(hbut only) → 教务系统验证(新增) → 查重 → bcrypt → 入库 → 返回token
```

**关键变更：**
- 参数校验后，检查 `schoolId`，仅允许 `"hbut"`
- 查重之前，调用 `services.VerifyHbutCredentials(req.StuID, req.Password)`
- 验证失败直接返回错误，不落库
- 其余逻辑不变（bcrypt + CreateUserWithPassword + 返回 JWT）

### 3. 新增检查接口

```
GET /api/v1/auth/check-user?stuId=xxx&schoolId=hbut
```

- 使用 `c.Query("stuId")` / `c.Query("schoolId")` 获取参数（不新增 DTO，参数少直接用 query）
- 调用 `models.GetUserByStuIDAndSchoolID` 查询
- 返回：

```json
// 用户存在
{ "success": true, "data": { "exists": true } }

// 用户不存在
{ "success": true, "data": { "exists": false } }
```

**路由注册（main.go）：** 放在公开 V1 路由组，与 register/login 同级，无需认证中间件。

### 5. 错误处理

| 场景 | HTTP 状态 | 消息 |
|------|-----------|------|
| 不支持的学校 | 400 | "暂不支持该学校" |
| 教务系统返回错误 | 400 | "学号或密码错误" |
| 教务系统不可达 | 500 | "教务系统验证失败，请稍后重试" |
| 密码加密失败 | 500 | "服务器错误" |
| 学号已注册 | 400 | "该学号在此学校已注册" |

## 依赖

- Go 标准库 `crypto/rsa`, `crypto/x509`, `encoding/base64` — RSA 加密
- `net/http` — 模拟登录请求
- 现有 `models.User`, `models.CreateUserWithPassword` — 数据持久化
- 现有 `services.GenerateToken`, `middleware.StudentAuth` — JWT 生成与验证

## 不纳入范围

- 不修改 `StudentLogin` 接口 — 登录仍然校验本地 bcrypt 密码
- 不支持 hbut 以外的学校 — 架构保留扩展点（`schoolId` 分发），但仅实现 hbut
- 不处理教务系统验证码（vcode/jcaptchaCode）— 当前 auth.js 也传空值
