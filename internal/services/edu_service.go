package services

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	captchaID = "fdHguSojgSJag5B74ij8Bu8ZAzWlNgXM"
	loginURL  = "https://jwxt.hbut.edu.cn/admin/login"
)

func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func uuid4() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// SchoolVerifier 教务系统验证函数类型
type SchoolVerifier func(stuID, password string) error

var schoolVerifiers = map[string]SchoolVerifier{
	"hbut": VerifyHbutCredentials,
}

func VerifySchoolCredentials(schoolID, stuID, password string) error {
	verifier, ok := schoolVerifiers[schoolID]
	if !ok {
		return errors.New("暂不支持该学校")
	}
	return verifier(stuID, password)
}

// VerifyHbutCredentials 模拟登录 HBUT 教务系统验证账号密码
// password 已经是前端 RSA 加密后的密文
func VerifyHbutCredentials(stuID, password string) error {
	if err := solveCaptchaAndLogin(stuID, password); err != nil {
		return err
	}
	return nil
}

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

// getCaptchaImages 获取验证码配置 + 图片 URL
func getCaptchaImages() (token, iv, shadeURL, cutoutURL string, err error) {
	now := time.Now().UnixMilli()
	confURL := fmt.Sprintf("https://captcha.chaoxing.com/captcha/get/conf?callback=cx_captcha_function&captchaId=%s&_=%d", captchaID, now)
	resp, err := http.Get(confURL)
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// 解析 JSONP: cx_captcha_function({...})
	start := strings.Index(bodyStr, "(")
	end := strings.LastIndex(bodyStr, ")")
	if start == -1 || end == -1 {
		return "", "", "", "", errors.New("解析 captcha config 失败")
	}
	var conf struct {
		T int64 `json:"t"`
	}
	if json.Unmarshal([]byte(bodyStr[start+1:end]), &conf) != nil {
		return "", "", "", "", errors.New("解析 captcha config JSON 失败")
	}

	t := strconv.FormatInt(conf.T, 10)
	captchaKey := md5Hash(t + uuid4())
	token = md5Hash(t + captchaID + "slide" + captchaKey) + ":" + strconv.FormatInt(conf.T+0x493e0, 10)
	iv = md5Hash(captchaID + "slide" + strconv.FormatInt(time.Now().UnixMilli(), 10) + uuid4())

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
}

// solveGap 调用云函数 captcha 计算缺口距离
func solveGap(shadeURL, cutoutURL string) (int, error) {
	result, err := Cloudbase.CallFunction("captcha", map[string]interface{}{
		"shadeImage":  shadeURL,
		"cutoutImage": cutoutURL,
	})
	if err != nil {
		return 0, err
	}
	if x, ok := result["x"].(float64); ok {
		return int(x), nil
	}
	return 0, errors.New("云函数未返回距离")
}

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
