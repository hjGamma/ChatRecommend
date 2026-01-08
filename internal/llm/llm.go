package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"ChatRecommend/internal/config"
	"ChatRecommend/internal/models"
	"github.com/sirupsen/logrus"
)

// Client 大模型客户端
type Client struct {
	config *config.LLMConfig
}

// Request 大模型请求
type Request struct {
	Context     string                 `json:"context"`
	Input       string                 `json:"input"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// Response 大模型响应
type Response struct {
	Text      string   `json:"text"`
	Suggestions []string `json:"suggestions,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// SummaryRequest 摘要生成请求
type SummaryRequest struct {
	Messages        []models.Message `json:"messages"`
	ExistingSummary *models.Summary  `json:"existing_summary,omitempty"`
	Config          map[string]interface{} `json:"config"`
}

// SummaryResponse 摘要生成响应
type SummaryResponse struct {
	Prompt  string                   `json:"prompt"`
	KeyInfo []map[string]interface{} `json:"key_info"`
	Error   string                   `json:"error,omitempty"`
}

// NewClient 创建大模型客户端
func NewClient(cfg *config.LLMConfig) *Client {
	return &Client{
		config: cfg,
	}
}

// Complete 生成补全建议
func (c *Client) Complete(context string, input string) ([]string, error) {
	req := Request{
		Context: context,
		Input:   input,
		Parameters: map[string]interface{}{
			"model":            c.config.API.Model,
			"temperature":      c.config.API.Temperature,
			"max_tokens":       c.config.API.MaxTokens,
			"top_p":            c.config.API.TopP,
			"frequency_penalty": c.config.API.FrequencyPenalty,
			"presence_penalty":  c.config.API.PresencePenalty,
		},
	}

	resp, err := c.callPython("complete", req)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("大模型返回错误: %s", resp.Error)
	}

	if len(resp.Suggestions) > 0 {
		return resp.Suggestions, nil
	}

	// 如果没有建议，从文本中提取
	if resp.Text != "" {
		return []string{resp.Text}, nil
	}

	return []string{}, nil
}

// GenerateSummary 生成对话摘要
func (c *Client) GenerateSummary(messages []models.Message, existingSummary *models.Summary) (string, string, error) {
	req := SummaryRequest{
		Messages:        messages,
		ExistingSummary: existingSummary,
		Config: map[string]interface{}{
			"max_summary_tokens": 500,
			"key_info_count":     10,
		},
	}

	resp, err := c.callPythonForSummary(req)
	if err != nil {
		return "", "", err
	}

	if resp.Error != "" {
		return "", "", fmt.Errorf("大模型返回错误: %s", resp.Error)
	}

	// 序列化关键信息
	keyInfoJSON := "[]"
	if len(resp.KeyInfo) > 0 {
		keyInfoBytes, err := json.Marshal(resp.KeyInfo)
		if err != nil {
			logrus.WithError(err).Warn("序列化关键信息失败")
		} else {
			keyInfoJSON = string(keyInfoBytes)
		}
	}

	return resp.Prompt, keyInfoJSON, nil
}

// callPython 调用Python脚本
func (c *Client) callPython(action string, req interface{}) (*Response, error) {
	reqJSON, err := json.Marshal(map[string]interface{}{
		"action": action,
		"request": req,
		"config": map[string]interface{}{
			"model_type": c.config.ModelType,
			"api":        c.config.API,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 执行Python脚本
	cmd := exec.Command(c.config.PythonInterpreter, c.config.PythonScript)
	cmd.Stdin = bytes.NewReader(reqJSON)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("执行Python脚本失败: %w, stderr: %s", err, stderr.String())
		}
	case <-time.After(time.Duration(c.config.Timeout) * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("调用大模型超时（%d秒）", c.config.Timeout)
	}

	// 解析响应
	var resp Response
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, stdout: %s", err, stdout.String())
	}

	return &resp, nil
}

// callPythonForSummary 调用Python脚本生成摘要
func (c *Client) callPythonForSummary(req SummaryRequest) (*SummaryResponse, error) {
	reqJSON, err := json.Marshal(map[string]interface{}{
		"action": "generate_summary",
		"request": req,
		"config": map[string]interface{}{
			"model_type": c.config.ModelType,
			"api":        c.config.API,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 执行Python脚本
	cmd := exec.Command(c.config.PythonInterpreter, c.config.PythonScript)
	cmd.Stdin = bytes.NewReader(reqJSON)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("执行Python脚本失败: %w, stderr: %s", err, stderr.String())
		}
	case <-time.After(time.Duration(c.config.Timeout) * time.Second):
		cmd.Process.Kill()
		return nil, fmt.Errorf("调用大模型超时（%d秒）", c.config.Timeout)
	}

	// 解析响应
	var resp SummaryResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, stdout: %s", err, stdout.String())
	}

	return &resp, nil
}

