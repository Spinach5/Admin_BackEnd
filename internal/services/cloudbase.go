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

func (c *CloudBaseClient) request(method, path string, body interface{}) (map[string]interface{}, error) {
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

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cloudbase 请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}
	return result, nil
}

func (c *CloudBaseClient) CallFunction(name string, data map[string]interface{}) (map[string]interface{}, error) {
	if data == nil {
		data = map[string]interface{}{}
	}
	path := fmt.Sprintf("/v1/functions/%s", name)
	return c.request("POST", path, data)
}

var Cloudbase = NewCloudBaseClient()
