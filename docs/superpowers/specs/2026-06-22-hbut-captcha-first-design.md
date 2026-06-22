# HBUT 注册：滑块验证优先 → 教务登录

## 目标

修改 HBUT 学校（schoolId="hbut"）的注册验证流程：每次先解滑块验证码（调用云函数计算缺口），再带 `jcaptchaCode` 登录教务系统。最多重试 3 次。

## 改动范围

单文件：`internal/services/edu_service.go`

## 现状

当前 `VerifyHbutCredentials` 先尝试直接 POST 登录，只有检测到验证码需求时才回退滑块流程。且获取滑块图片用 GET（应为 POST），提交验证后未提取 `validate`，登录时未传 `jcaptchaCode`。

## 目标行为

```
StudentRegister handler
  └─ VerifySchoolCredentials("hbut", stuID, password)
       └─ VerifyHbutCredentials(stuID, password)       // 简化：直接走滑块
            └─ solveCaptchaAndLogin(stuID, password)    // 循环最多 3 次
                 ├─ getCaptchaImages()  → token, iv, shadeURL, cutoutURL
                 ├─ solveGap(shade, cut) → x (调用云函数 captcha)
                 ├─ submitCaptcha(token, iv, x) → validate
                 └─ postLogin(stuID, pwd, validate)
                      ├─ 302 → 成功，返回 nil
                      └─ 非 302 → 继续下一次循环
```

## 具体变更

### 1. `VerifyHbutCredentials` — 简化

去掉"先直接登录"的分支，直接调用 `solveCaptchaAndLogin`。失败返回 `"验证码验证失败，请手动登录教务系统"`。

### 2. `solveCaptchaAndLogin` — 重写主循环

```
循环 3 次:
  1. getCaptchaImages() → token, iv, shadeURL, cutoutURL
  2. solveGap(shadeURL, cutoutURL) → x (调用 Cloudbase.CallFunction("captcha", ...))
     若 x < 10 → continue
  3. submitCaptcha(token, iv, x) → validate
     若失败 → continue
  4. postLogin(stuID, encPwd, validate) → status
     若 302 → return nil (成功)
     否则 continue
3 次结束 → return error
```

### 3. `getCaptchaImages` — 图片接口改 POST

- 请求方式从 `http.Get` 改为 `http.Post`（`application/x-www-form-urlencoded`）
- body 增加 `jcaptchaDefect=1`（与参考 JS 一致）
- 返回值：从 `(token, iv, shadeURL, cutoutURL, error)` 保持不变，token 对应图片响应中的 verifyToken

### 4. `submitCaptcha` — 返回 validate

- 签名从 `func(token, iv string, x int) (bool, error)` 改为 `func(token, iv string, x int) (string, error)`
- 从响应的 `extraData` JSON 字符串中解析 `validate` 字段
- 解析失败时回退到正则匹配 `"validate":"..."` 
- 返回 validate 字符串

### 5. `postLogin` — 增加 jcaptchaCode 参数

- 签名改为 `func(stuID, encPwd, jcaptchaCode string) (int, string, error)`
- jcaptchaCode 非空时，form 中加入 `jcaptchaCode=validate`

## 不影响

- `Cloudbase.CallFunction("captcha", ...)` 调用方式不变
- `SolveGap` 函数不变（仍通过云函数求解）
- Handler 层（`v1_auth.go`）不变，`dto` 不变
- 其他学校（非 hbut）的验证逻辑不受影响
- `/api/internal/captcha-solve` 端点不变（留给其他用途）
