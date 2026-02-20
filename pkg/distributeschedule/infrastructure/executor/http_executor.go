package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bamboo/pkg/distributeschedule/domain/model"
	"bamboo/pkg/distributeschedule/domain/service"
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

func (e *HTTPExecutor) Execute(ctx context.Context, task *model.Task) (*model.TaskResult, error) {
	// 从 payload 中解析 HTTP 请求参数
	payload, ok := task.Result.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload format")
	}

	url, _ := payload["url"].(string)
	method, _ := payload["method"].(string)
	body, _ := payload["body"]

	if url == "" {
		return nil, fmt.Errorf("url is required")
	}
	if method == "" {
		method = "GET"
	}

	// 构建请求
	var reqBody io.Reader
	if body != nil {
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

	result := &model.TaskResult{
		Code:    resp.StatusCode,
		Message: resp.Status,
		Data:    string(respBody),
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("http request failed: %s", resp.Status)
	}

	return result, nil
}

func (e *HTTPExecutor) Type() string {
	return "http"
}

func (e *HTTPExecutor) Protocol() string {
	return "http"
}
