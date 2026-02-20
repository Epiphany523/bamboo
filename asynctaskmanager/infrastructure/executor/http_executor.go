package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bamboo/asynctaskmanager/domain/model"
	"bamboo/asynctaskmanager/domain/service"
)

// HTTPExecutor HTTP 执行器
type HTTPExecutor struct {
	client *http.Client
}

// NewHTTPExecutor 创建 HTTP 执行器
func NewHTTPExecutor() service.Executor {
	return &HTTPExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (e *HTTPExecutor) Execute(ctx context.Context, task *model.Task) (map[string]interface{}, error) {
	// 从 payload 中解析 HTTP 请求参数
	url, ok := task.Payload["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url is required in payload")
	}

	method := "POST"
	if m, ok := task.Payload["method"].(string); ok {
		method = m
	}

	// 构建请求
	var reqBody io.Reader
	if body, ok := task.Payload["body"]; ok {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body failed: %w", err)
		}
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	result := map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("http request failed: %s", resp.Status)
	}

	return result, nil
}

func (e *HTTPExecutor) Type() model.ExecutorType {
	return model.ExecutorTypeHTTP
}

func (e *HTTPExecutor) SupportedTaskTypes() []string {
	return []string{"http_request", "webhook"}
}
