# HBUT Captcha-First Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change HBUT school registration to solve slider captcha first (before login), calling the cloud function for gap computation, then pass `jcaptchaCode` validate token to the educational system login. Max 3 retries.

**Architecture:** All changes are within `internal/services/edu_service.go`. Five functions are modified bottom-up: `getCaptchaImages` (GET→POST, add `jcaptchaDefect`, return verifyToken), `submitCaptcha` (return validate string), `postLogin` (add `jcaptchaCode` param), `solveCaptchaAndLogin` (always solve captcha first, pass validate to login), `VerifyHbutCredentials` (remove direct login attempt).

**Tech Stack:** Go 1.x, standard library (`net/http`, `net/url`), existing `Cloudbase.CallFunction`

## Global Constraints

- Only modify `internal/services/edu_service.go`
- Cloud function call via `Cloudbase.CallFunction("captcha", ...)` — unchanged
- Max 3 captcha attempts per login
- Must pass `jcaptchaCode` (validate value) in login form
- Existing `dto`, `handlers`, `config` packages — zero changes

---

### Task 1: Change `getCaptchaImages` — POST + `jcaptchaDefect` + return verifyToken

**Files:**
- Modify: `internal/services/edu_service.go` (function `getCaptchaImages`)

**What changes:**
1. Switch image fetch from GET to POST, adding `jcaptchaDefect=1` to the form body
2. Parse `token` from the image JSONP response (the verifyToken) and return it instead of the original computed token
3. Signature unchanged: `(token, iv, shadeURL, cutoutURL string, err error)`

- [ ] **Step 1: Replace the image fetch and response parsing**

Find the block starting at `imgParams := url.Values{}` in `getCaptchaImages` and replace through the return statement.

Old code:

```go
	imgParams := url.Values{}
	imgParams.Set("callback", "cx_captcha_function")
	imgParams.Set("captchaId", captchaID)
	imgParams.Set("type", "slide")
	imgParams.Set("version", "1.1.20")
	imgParams.Set("captchaKey", captchaKey)
	imgParams.Set("token", token)
	imgParams.Set("referer", loginURL)
	imgParams.Set("iv", iv)
	imgParams.Set("_", strconv.FormatInt(time.Now().UnixMilli(), 10))

	imgURL := fmt.Sprintf("https://captcha.chaoxing.com/captcha/get/verification/image?%s", imgParams.Encode())
	imgResp, err := http.Get(imgURL)
	if err != nil {
		return "", "", "", "", err
	}
	defer imgResp.Body.Close()
	imgBody, _ := io.ReadAll(imgResp.Body)
	imgStr := string(imgBody)

	start = strings.Index(imgStr, "(")
	end = strings.LastIndex(imgStr, ")")
	if start == -1 || end == -1 {
		return "", "", "", "", errors.New("解析 captcha image 失败")
	}
	var imgData struct {
		ImageVerificationVo struct {
			ShadeImage  string `json:"shadeImage"`
			CutoutImage string `json:"cutoutImage"`
		} `json:"imageVerificationVo"`
	}
	if json.Unmarshal([]byte(imgStr[start+1:end]), &imgData) != nil {
		return "", "", "", "", errors.New("解析 captcha image JSON 失败")
	}

	return token, iv, imgData.ImageVerificationVo.ShadeImage, imgData.ImageVerificationVo.CutoutImage, nil
```

Replace with:

```go
	imgParams := url.Values{}
	imgParams.Set("callback", "cx_captcha_function")
	imgParams.Set("captchaId", captchaID)
	imgParams.Set("type", "slide")
	imgParams.Set("version", "1.1.20")
	imgParams.Set("captchaKey", captchaKey)
	imgParams.Set("token", token)
	imgParams.Set("referer", loginURL)
	imgParams.Set("jcaptchaDefect", "1")
	imgParams.Set("iv", iv)
	imgParams.Set("_", strconv.FormatInt(time.Now().UnixMilli(), 10))

	imgReq, err := http.NewRequest("POST",
		"https://captcha.chaoxing.com/captcha/get/verification/image",
		strings.NewReader(imgParams.Encode()))
	if err != nil {
		return "", "", "", "", err
	}
	imgReq.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	imgResp, err := http.DefaultClient.Do(imgReq)
	if err != nil {
		return "", "", "", "", err
	}
	defer imgResp.Body.Close()
	imgBody, _ := io.ReadAll(imgResp.Body)
	imgStr := string(imgBody)

	start = strings.Index(imgStr, "(")
	end = strings.LastIndex(imgStr, ")")
	if start == -1 || end == -1 {
		return "", "", "", "", errors.New("解析 captcha image 失败")
	}
	var imgData struct {
		Token               string `json:"token"`
		ImageVerificationVo struct {
			ShadeImage  string `json:"shadeImage"`
			CutoutImage string `json:"cutoutImage"`
		} `json:"imageVerificationVo"`
	}
	if json.Unmarshal([]byte(imgStr[start+1:end]), &imgData) != nil {
		return "", "", "", "", errors.New("解析 captcha image JSON 失败")
	}

	// 返回图片响应中的 verifyToken（用于后续 submitCaptcha），而非原始 token
	verifyToken := imgData.Token
	if verifyToken == "" {
		return "", "", "", "", errors.New("图片响应中未获取到 verifyToken")
	}
	return verifyToken, iv, imgData.ImageVerificationVo.ShadeImage, imgData.ImageVerificationVo.CutoutImage, nil
```

- [ ] **Step 2: Build to verify compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: builds successfully.

- [ ] **Step 3: Commit**

```bash
git add internal/services/edu_service.go
git commit -m "fix: change captcha image fetch from GET to POST, return verifyToken

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: Change `submitCaptcha` — return validate string

**Files:**
- Modify: `internal/services/edu_service.go` (function `submitCaptcha`)

**What changes:** The function currently returns `(bool, error)`. Change it to return `(string, error)` where the string is the `validate` value extracted from the captcha verification response. Parse `extraData.validate` from the JSONP response, with a regex fallback.

- [ ] **Step 1: Replace `submitCaptcha` function**

Replace the entire function with:

```go
// submitCaptcha 提交验证码结果到超星，返回 validate 字符串
func submitCaptcha(token, iv string, x int) (string, error) {
	params := url.Values{}
	params.Set("callback", "cx_captcha_function")
	params.Set("captchaId", captchaID)
	params.Set("type", "slide")
	params.Set("token", token)
	params.Set("textClickArr", fmt.Sprintf(`[{"x":%d}]`, x))
	params.Set("coordinate", "[]")
	params.Set("runEnv", "10")
	params.Set("version", "1.1.20")
	params.Set("t", "a")
	params.Set("iv", iv)
	params.Set("_", strconv.FormatInt(time.Now().UnixMilli(), 10))

	checkURL := fmt.Sprintf("https://captcha.chaoxing.com/captcha/check/verification/result?%s", params.Encode())
	resp, err := http.Get(checkURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	start := strings.Index(bodyStr, "(")
	end := strings.LastIndex(bodyStr, ")")
	if start == -1 || end == -1 {
		return "", errors.New("解析 captcha result 失败")
	}

	var result struct {
		Result    bool        `json:"result"`
		Code      interface{} `json:"code"`
		ExtraData interface{} `json:"extraData"`
	}
	if err := json.Unmarshal([]byte(bodyStr[start+1:end]), &result); err != nil {
		return "", fmt.Errorf("解析 captcha result JSON 失败: %w", err)
	}

	code := fmt.Sprintf("%v", result.Code)
	if code != "0" && code != "200" && !result.Result {
		return "", fmt.Errorf("captcha 验证失败: code=%s, result=%v", code, result.Result)
	}

	// 提取 validate: 优先从 extraData JSON 字符串解析
	if ed, ok := result.ExtraData.(string); ok && ed != "" {
		var extra struct {
			Validate string `json:"validate"`
		}
		if json.Unmarshal([]byte(ed), &extra) == nil && extra.Validate != "" {
			return extra.Validate, nil
		}
	}

	// 回退: 从原始响应正则匹配
	if idx := strings.Index(bodyStr, `"validate":"`); idx != -1 {
		vs := idx + len(`"validate":"`)
		if ve := strings.Index(bodyStr[vs:], `"`); ve != -1 {
			validate := bodyStr[vs : vs+ve]
			validate = strings.ReplaceAll(validate, `\`, "")
			return validate, nil
		}
	}

	return "", errors.New("captcha 验证成功但未提取到 validate")
}
```

- [ ] **Step 2: Build to verify compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: **FAIL** — `solveCaptchaAndLogin` still expects `(bool, error)` from `submitCaptcha`. Expected; Task 4 will fix it.

- [ ] **Step 3: Commit**

```bash
git add internal/services/edu_service.go
git commit -m "fix: change submitCaptcha to return validate string

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: Add `jcaptchaCode` parameter to `postLogin`

**Files:**
- Modify: `internal/services/edu_service.go` (function `postLogin`)

**What changes:** Add a `jcaptchaCode string` parameter. When non-empty, add it to the login form data.

- [ ] **Step 1: Replace `postLogin` function**

Replace the entire function with:

```go
// postLogin POST 教务登录，返回状态码和响应体
func postLogin(stuID, encPwd, jcaptchaCode string) (int, string, error) {
	form := url.Values{}
	form.Set("username", stuID)
	form.Set("password", encPwd)
	form.Set("rememberMe", "1")
	if jcaptchaCode != "" {
		form.Set("jcaptchaCode", jcaptchaCode)
	}

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Referer", "https://jwxt.hbut.edu.cn")
	req.Header.Set("Origin", "https://jwxt.hbut.edu.cn")

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("教务系统验证失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body), nil
}
```

- [ ] **Step 2: Build to verify compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: **FAIL** — `solveCaptchaAndLogin` and `VerifyHbutCredentials` still call `postLogin` with 2 args. Tasks 4 and 5 will fix this.

- [ ] **Step 3: Commit**

```bash
git add internal/services/edu_service.go
git commit -m "feat: add jcaptchaCode parameter to postLogin

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: Rewrite `solveCaptchaAndLogin` — always captcha-first with validate

**Files:**
- Modify: `internal/services/edu_service.go` (function `solveCaptchaAndLogin`)

**What changes:** Remove the old conditional logic. Always: get captcha images → solve gap via cloud function → submit captcha to get validate → login with `jcaptchaCode`. Loop up to 3 times. Return `error` instead of `bool`.

- [ ] **Step 1: Replace `solveCaptchaAndLogin` function**

Replace the entire function with:

```go
// solveCaptchaAndLogin 先解滑块验证码（最多3次），然后带 jcaptchaCode 登录教务系统
func solveCaptchaAndLogin(stuID, encPwd string) error {
	for attempt := 0; attempt < 3; attempt++ {
		// 1. 获取验证码图片
		token, iv, shadeURL, cutoutURL, err := getCaptchaImages()
		if err != nil {
			log.Printf("[Captcha] 获取验证码图片失败: %v", err)
			continue
		}

		// 2. 云函数求解缺口距离
		x, err := solveGap(shadeURL, cutoutURL)
		if err != nil {
			log.Printf("[Captcha] 求解失败: %v", err)
			continue
		}
		log.Printf("[Captcha] 缺口距离: %dpx", x)
		if x < 10 {
			continue
		}

		// 3. 提交验证码结果，获取 validate
		validate, err := submitCaptcha(token, iv, x)
		if err != nil {
			log.Printf("[Captcha] 提交验证码失败: %v", err)
			continue
		}
		log.Printf("[Captcha] solved, validate: %s", validate)

		// 4. 带 jcaptchaCode 登录教务系统
		status, _, err := postLogin(stuID, encPwd, validate)
		if err != nil {
			log.Printf("[Captcha] 登录请求失败: %v", err)
			continue
		}
		if status >= 300 && status < 400 {
			return nil
		}
		log.Printf("[Captcha] 带验证码登录失败: status=%d", status)
	}
	return errors.New("验证码验证失败，请手动登录教务系统 https://jwxt.hbut.edu.cn 后重试")
}
```

- [ ] **Step 2: Build to verify compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: **FAIL** — `VerifyHbutCredentials` still calls `postLogin` with 2 args and expects `bool` from `solveCaptchaAndLogin`. Task 5 will fix this.

- [ ] **Step 3: Commit**

```bash
git add internal/services/edu_service.go
git commit -m "feat: rewrite solveCaptchaAndLogin to always solve captcha first

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 5: Simplify `VerifyHbutCredentials` — remove direct login attempt

**Files:**
- Modify: `internal/services/edu_service.go` (function `VerifyHbutCredentials`)

**What changes:** Remove the "try direct login first, fall back to captcha" logic. Just call `solveCaptchaAndLogin` directly.

- [ ] **Step 1: Replace `VerifyHbutCredentials` function**

Replace the entire function with:

```go
// VerifyHbutCredentials 模拟登录 HBUT 教务系统验证账号密码
// password 已经是前端 RSA 加密后的密文
func VerifyHbutCredentials(stuID, password string) error {
	if err := solveCaptchaAndLogin(stuID, password); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 2: Build to verify full compilation**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

Expected: **PASS** — builds successfully, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/services/edu_service.go
git commit -m "refactor: simplify VerifyHbutCredentials to always use captcha-first flow

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 6: Final verification

- [ ] **Step 1: Build and vet**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./... && go vet ./internal/services/
```

Expected: builds and passes vet with no errors.

- [ ] **Step 2: Review the complete changed file**

Read `internal/services/edu_service.go` and verify:
- `VerifyHbutCredentials` calls only `solveCaptchaAndLogin`
- `solveCaptchaAndLogin` loops 3 times: getCaptchaImages → solveGap → submitCaptcha → postLogin
- `getCaptchaImages` uses POST with `jcaptchaDefect=1`, returns verifyToken from image response
- `submitCaptcha` returns `(string, error)` — the validate value parsed from extraData/regex
- `postLogin` accepts `jcaptchaCode string` and adds it to form when non-empty

- [ ] **Step 3: Check for unused imports**

```bash
cd /home/zqw/biancheng/Project/backend && go build ./...
```

If it builds, imports are fine.
