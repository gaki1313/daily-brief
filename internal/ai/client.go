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
	promptBuilder.WriteString("请根据以下信息生成一份自然、简洁的工作日报，语言要像人工写的一样自然，避免过于AI化的表达。\n\n")

	// 要求和格式
	promptBuilder.WriteString("写作要求：\n")
	promptBuilder.WriteString("1. 语言自然、简洁，像人写的一样\n")
	promptBuilder.WriteString("2. 避免过于正式或AI化的表达\n")
	promptBuilder.WriteString("3. 重点突出，条理清晰\n")
	promptBuilder.WriteString("4. 直接描述做了什么，不要过度包装\n\n")

	// 输出格式
	promptBuilder.WriteString("格式示例：\n")
	promptBuilder.WriteString("# " + date + " 工作日报\n\n")
	promptBuilder.WriteString("## 今日工作\n")
	promptBuilder.WriteString("1. [具体做了什么事情]\n")
	promptBuilder.WriteString("2. [具体做了什么事情]\n\n")
	promptBuilder.WriteString("## 代码提交\n")
	promptBuilder.WriteString("- [提交的具体内容，用简单的话描述]\n\n")
	promptBuilder.WriteString("## 其他\n")
	promptBuilder.WriteString("- [其他工作内容]\n\n")

	// 用户自定义工作成果
	if len(workItems) > 0 {
		promptBuilder.WriteString("今日完成的工作：\n")
		for i, item := range workItems {
			// 移除工作项目开头的数字编号，避免重复编号
			cleanItem := strings.TrimSpace(item)
			// 如果以数字开头，移除数字和点号
			if len(cleanItem) > 0 && cleanItem[0] >= '0' && cleanItem[0] <= '9' {
				if dotIndex := strings.Index(cleanItem, "."); dotIndex > 0 && dotIndex < 5 {
					cleanItem = strings.TrimSpace(cleanItem[dotIndex+1:])
				}
			}
			promptBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, cleanItem))
		}
		promptBuilder.WriteString("\n")
	}

	// Git提交记录
	promptBuilder.WriteString("代码提交记录：\n")
	if len(commits) == 0 {
		promptBuilder.WriteString("今日无代码提交\n")
	} else {
		for i, commit := range commits {
			promptBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, commit.Message))
			if commit.FilesChanged > 0 {
				promptBuilder.WriteString(fmt.Sprintf("   涉及%d个文件，+%d -%d行代码\n",
					commit.FilesChanged, commit.Additions, commit.Deletions))
			}
		}
	}

	promptBuilder.WriteString("\n重要要求：\n")
	promptBuilder.WriteString("1. 必须包含「今日完成的工作」中的所有内容\n")
	promptBuilder.WriteString("2. 直接描述工作成果，不要提及「用户指定」或类似表述\n")
	promptBuilder.WriteString("3. 不要包含任何处理过程或元信息的描述\n")
	promptBuilder.WriteString("4. 语言简洁专业，避免「嗯」「对了」等口语化表达\n")
	promptBuilder.WriteString("5. 不要添加「重要说明」等额外的解释性段落\n\n")
	promptBuilder.WriteString("请生成一份简洁专业的工作日报：\n")

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
