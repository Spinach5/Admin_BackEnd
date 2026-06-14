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
	// 1. 直接 POST 登录
	status, body, err := postLogin(stuID, password)
	if err != nil {
		return err
	}

	// 302 = 成功
	if status >= 300 && status < 400 {
		return nil
	}

	// JSON = 错误
	if strings.HasPrefix(body, "{") {
		var result struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Msg     string `json:"msg"`
		}
		if json.Unmarshal([]byte(body), &result) == nil && result.Code != 0 && result.Code != 200 {
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

	// 验证码检测
	if strings.Contains(body, "captcha") || strings.Contains(body, "jcaptchaCode") ||
		strings.Contains(body, "chaoxing.com/load.min.js") {
		if solveCaptchaAndLogin(stuID, password) {
			return nil
		}
		return errors.New("验证码验证失败，请手动登录教务系统 https://jwxt.hbut.edu.cn 后重试")
	}

	return nil
}

// postLogin POST 教务登录，返回状态码和响应体
func postLogin(stuID, encPwd string) (int, string, error) {
	form := url.Values{}
	form.Set("username", stuID)
	form.Set("password", encPwd)
	form.Set("rememberMe", "1")

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

// solveCaptchaAndLogin 获取验证码图片 → 云函数求解 → 提交 → 重新登录
func solveCaptchaAndLogin(stuID, encPwd string) bool {
	for attempt := 0; attempt < 3; attempt++ {
		// 1. 获取验证码配置和图片 URL
		token, iv, shadeURL, cutoutURL, err := getCaptchaImages()
		if err != nil {
			log.Printf("[Captcha] 获取验证码图片失败: %v", err)
			continue
		}

		// 2. 调用云函数求解缺口距离
		x, err := solveGap(shadeURL, cutoutURL)
		if err != nil {
			log.Printf("[Captcha] 求解失败: %v", err)
			continue
		}
		log.Printf("[Captcha] 缺口距离: %dpx", x)
		if x < 10 {
			continue
		}

		// 3. 提交验证码结果
		ok, err := submitCaptcha(token, iv, x)
		if err != nil || !ok {
			log.Printf("[Captcha] 提交验证码失败: %v", err)
			continue
		}

		// 4. 重新登录
		status, _, err := postLogin(stuID, encPwd)
		if err != nil {
			continue
		}
		if status >= 300 && status < 400 {
			return true
		}
		log.Printf("[Captcha] 带验证码登录仍失败: status=%d", status)
	}
	return false
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

// submitCaptcha 提交验证码结果到 超星
func submitCaptcha(token, iv string, x int) (bool, error) {
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
		return false, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	start := strings.Index(bodyStr, "(")
	end := strings.LastIndex(bodyStr, ")")
	if start == -1 || end == -1 {
		return false, errors.New("解析 captcha result 失败")
	}
	var result struct {
		Code interface{} `json:"code"`
	}
	json.Unmarshal([]byte(bodyStr[start+1:end]), &result)

	code := fmt.Sprintf("%v", result.Code)
	return code == "0" || code == "200", nil
}
