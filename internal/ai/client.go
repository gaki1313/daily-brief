// Package ai 提供AI智能生成功能
// 支持多种AI提供商：Ollama、OpenAI、Gemini、通义千问等
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"daily-brief/internal/config"
	"daily-brief/internal/storage"

	"github.com/sirupsen/logrus"
)

// Client AI客户端
type Client struct {
	config config.AIConfig
	logger *logrus.Logger
	client *http.Client
}

// NewClient 创建新的AI客户端
func NewClient(cfg config.AIConfig, logger *logrus.Logger) *Client {
	return &Client{
		config: cfg,
		logger: logger,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

// IsEnabled 检查AI是否启用
func (c *Client) IsEnabled() bool {
	return c.config.Enabled
}

// GenerateReport 生成AI报告
func (c *Client) GenerateReport(ctx context.Context, commits []storage.Commit, date string, style string, workItems []string) (string, error) {
	if !c.config.Enabled {
		return "", fmt.Errorf("AI is not enabled")
	}

	// 构建提示词
	prompt := c.buildPrompt(commits, date, style, workItems)

	// 根据提供商调用不同的API
	switch strings.ToLower(c.config.Provider) {
	case "ollama":
		return c.generateWithOllama(ctx, prompt)
	case "openai":
		return c.generateWithOpenAI(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported AI provider: %s", c.config.Provider)
	}
}

// buildPrompt 构建提示词
func (c *Client) buildPrompt(commits []storage.Commit, date string, style string, workItems []string) string {
	var promptBuilder strings.Builder

	// 系统提示
	promptBuilder.WriteString("你是一个专业的工作日报生成助手。请根据以下Git提交记录和用户提供的工作成果，生成一份结构化的中文工作日报。\n\n")

	// 要求和格式
	promptBuilder.WriteString("要求：\n")
	promptBuilder.WriteString("1. 使用中文回复\n")
	promptBuilder.WriteString("2. 内容要专业、简洁、有条理\n")
	promptBuilder.WriteString("3. 突出重要的功能开发、bug修复、性能优化\n")
	promptBuilder.WriteString("4. 提取业务价值和技术亮点\n")
	promptBuilder.WriteString("5. 按重要性排序工作项\n")
	promptBuilder.WriteString("6. 将用户自定义工作成果与Git提交记录相结合\n\n")

	// 输出格式
	promptBuilder.WriteString("输出格式：\n")
	promptBuilder.WriteString("# 工作日报 - " + date + "\n\n")
	promptBuilder.WriteString("## 工作概述\n")
	promptBuilder.WriteString("[简要总结当天的主要工作内容，包括代码开发和其他工作]\n\n")
	promptBuilder.WriteString("## 具体工作项\n")
	promptBuilder.WriteString("### 功能开发\n")
	promptBuilder.WriteString("- [功能开发相关的工作]\n\n")
	promptBuilder.WriteString("### Bug修复\n")
	promptBuilder.WriteString("- [Bug修复相关的工作]\n\n")
	promptBuilder.WriteString("### 优化改进\n")
	promptBuilder.WriteString("- [性能优化、代码改进等]\n\n")
	promptBuilder.WriteString("### 其他工作\n")
	promptBuilder.WriteString("- [非代码相关的工作成果]\n\n")
	promptBuilder.WriteString("## 技术亮点\n")
	promptBuilder.WriteString("- [技术相关的重要改进或创新]\n\n")
	promptBuilder.WriteString("## 工作量统计\n")
	promptBuilder.WriteString("- 提交次数：[数量]\n")
	promptBuilder.WriteString("- 代码变更：+[新增行数] -[删除行数]\n")
	promptBuilder.WriteString("- 涉及文件：[文件数量]个\n\n")

	// 用户自定义工作成果
	if len(workItems) > 0 {
		promptBuilder.WriteString("用户自定义工作成果：\n")
		for i, item := range workItems {
			promptBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		}
		promptBuilder.WriteString("\n")
	}

	// Git提交记录
	promptBuilder.WriteString("Git提交记录：\n")
	if len(commits) == 0 {
		promptBuilder.WriteString("今日无提交记录\n")
	} else {
		for i, commit := range commits {
			promptBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, commit.Message))
			if commit.FilesChanged > 0 {
				promptBuilder.WriteString(fmt.Sprintf("   文件变更：%d个文件，+%d -%d行\n",
					commit.FilesChanged, commit.Additions, commit.Deletions))
			}
		}
	}

	promptBuilder.WriteString("\n请基于以上信息生成专业的工作日报，确保将用户自定义工作成果与Git提交记录有机结合：\n")

	return promptBuilder.String()
}

// generateWithOllama 使用Ollama生成
func (c *Client) generateWithOllama(ctx context.Context, prompt string) (string, error) {
	url := c.config.OllamaURL + "/api/generate"

	requestBody := map[string]interface{}{
		"model":  c.config.OllamaModel,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": c.config.Temperature,
			"num_predict": c.config.MaxTokens,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"url":   url,
		"model": c.config.OllamaModel,
	}).Debug("Sending request to Ollama")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	if !response.Done {
		return "", fmt.Errorf("Ollama generation not completed")
	}

	c.logger.WithField("response_length", len(response.Response)).Info("Ollama generation completed")

	return response.Response, nil
}

// generateWithOpenAI 使用OpenAI生成
func (c *Client) generateWithOpenAI(ctx context.Context, prompt string) (string, error) {
	url := c.config.OpenAIURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}

	requestBody := map[string]interface{}{
		"model": c.config.OpenAIModel,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": c.config.Temperature,
		"max_tokens":  c.config.MaxTokens,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.OpenAIKey)

	c.logger.WithFields(logrus.Fields{
		"url":   url,
		"model": c.config.OpenAIModel,
	}).Debug("Sending request to OpenAI")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	content := response.Choices[0].Message.Content
	c.logger.WithField("response_length", len(content)).Info("OpenAI generation completed")

	return content, nil
}

// TestConnection 测试AI连接
func (c *Client) TestConnection(ctx context.Context) error {
	if !c.config.Enabled {
		return fmt.Errorf("AI is not enabled")
	}

	testPrompt := "请回复：AI连接测试成功"

	switch strings.ToLower(c.config.Provider) {
	case "ollama":
		return c.testOllamaConnection(ctx, testPrompt)
	case "openai":
		return c.testOpenAIConnection(ctx, testPrompt)
	default:
		return fmt.Errorf("unsupported AI provider: %s", c.config.Provider)
	}
}

// testOllamaConnection 测试Ollama连接
func (c *Client) testOllamaConnection(ctx context.Context, prompt string) error {
	url := c.config.OllamaURL + "/api/generate"

	requestBody := map[string]interface{}{
		"model":  c.config.OllamaModel,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": 50,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama test failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// testOpenAIConnection 测试OpenAI连接
func (c *Client) testOpenAIConnection(ctx context.Context, prompt string) error {
	url := c.config.OpenAIURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}

	requestBody := map[string]interface{}{
		"model": c.config.OpenAIModel,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 50,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal test request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.OpenAIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to OpenAI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenAI test failed: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}
