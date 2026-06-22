package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type CloudBaseClient struct {
	EnvID       string
	AccessToken string
	BaseURL     string
	HTTPClient  *http.Client
}

func NewCloudBaseClient() *CloudBaseClient {
	godotenv.Load()

	envID := os.Getenv("CLOUDBASE_ENV_ID")
	accessToken := os.Getenv("CLOUDBASE_ACCESS_TOKEN")

	return &CloudBaseClient{
		EnvID:       envID,
		AccessToken: accessToken,
		BaseURL:     fmt.Sprintf("https://%s.api.tcloudbasegateway.com", envID),
		HTTPClient:  &http.Client{},
	}
}

func (c *CloudBaseClient) Request(method, path string, body any, customHeaders map[string]string) (any, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("JSON序列化失败: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	for key, value := range customHeaders {
		req.Header.Set(key, value)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if len(bodyBytes) == 0 {
		return true, nil
	}

	var result any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	return result, nil
}

var Cloudbase = NewCloudBaseClient()
