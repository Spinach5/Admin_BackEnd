package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SchoolVerifier 教务系统验证函数类型
type SchoolVerifier func(stuID, password string) error

// schoolVerifiers 支持的学校及其验证函数
var schoolVerifiers = map[string]SchoolVerifier{
	"hbut": VerifyHbutCredentials,
}

// VerifySchoolCredentials 根据学校代码调用对应的教务验证函数
// 成功返回 nil，不支持该学校或验证失败返回 error
func VerifySchoolCredentials(schoolID, stuID, password string) error {
	verifier, ok := schoolVerifiers[schoolID]
	if !ok {
		return errors.New("暂不支持该学校")
	}
	return verifier(stuID, password)
}

// VerifyHbutCredentials 模拟登录 HBUT 教务系统验证账号密码
// password 参数已经是前端 RSA 加密后的密文，直接转发，不再二次加密
// 成功返回 nil，失败返回 error
func VerifyHbutCredentials(stuID, password string) error {
	// 构造 form 请求体（密码已是前端加密后的密文）
	form := url.Values{}
	form.Set("username", stuID)
	form.Set("password", password)
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

	// 4. 判断结果：返回 JSON 表示失败（参考 auth.js 逻辑）
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
