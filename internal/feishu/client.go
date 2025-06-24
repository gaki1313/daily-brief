// Package feishu 飞书集成模块
// 提供飞书API调用、消息发送、群聊管理等功能
package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"daily-brief/internal/config"
	"daily-brief/internal/storage"

	"github.com/sirupsen/logrus"
)

// Client 飞书API客户端
// 封装飞书开放平台API调用
type Client struct {
	config     config.FeishuConfig  // 飞书配置
	database   storage.Database     // 数据库接口
	logger     *logrus.Logger       // 日志器
	httpClient *http.Client         // HTTP客户端
	
	// 认证信息
	accessToken string    // 访问令牌
	tokenExpiry time.Time // 令牌过期时间
}

// MessageType 消息类型枚举
type MessageType string

const (
	MessageTypeText        MessageType = "text"        // 文本消息
	MessageTypeRichText    MessageType = "rich_text"   // 富文本消息
	MessageTypeMarkdown    MessageType = "markdown"    // Markdown消息
	MessageTypeCard        MessageType = "interactive" // 卡片消息
)

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	ReceiveID   string            `json:"receive_id"`            // 接收者ID
	MessageType MessageType       `json:"msg_type"`              // 消息类型
	Content     string            `json:"content"`               // 消息内容
	UUID        string            `json:"uuid,omitempty"`        // 消息UUID（防重复）
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	Code int    `json:"code"`    // 错误码
	Msg  string `json:"msg"`     // 错误信息
	Data struct {
		MessageID string `json:"message_id"` // 消息ID
	} `json:"data"`
}

// AccessTokenResponse 访问令牌响应
type AccessTokenResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AccessToken string `json:"access_token"`
		Expire      int    `json:"expire"`
	} `json:"data"`
}

// NewClient 创建新的飞书客户端
func NewClient(cfg config.FeishuConfig, db storage.Database, logger *logrus.Logger) *Client {
	return &Client{
		config:   cfg,
		database: db,
		logger:   logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression: false,
			},
		},
	}
}

// IsEnabled 检查飞书集成是否启用
func (c *Client) IsEnabled() bool {
	if !c.config.Enabled {
		return false
	}
	// API模式需要AppID和AppSecret
	if c.config.AppID != "" && c.config.AppSecret != "" {
		return true
	}
	// Webhook模式只需要WebhookURL
	if c.config.WebhookURL != "" {
		return true
	}
	return false
}

// getAccessToken 获取访问令牌
// 实现自动刷新机制
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	// 检查当前令牌是否有效
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	// 请求新的访问令牌
	reqBody := map[string]string{
		"app_id":     c.config.AppID,
		"app_secret": c.config.AppSecret,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", 
		"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal", 
		bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request access token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Code != 0 {
		return "", fmt.Errorf("failed to get access token: %s", tokenResp.Msg)
	}

	// 更新令牌信息
	c.accessToken = tokenResp.Data.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.Data.Expire-60) * time.Second) // 提前1分钟过期

	c.logger.WithFields(logrus.Fields{
		"expires_in": tokenResp.Data.Expire,
		"expires_at": c.tokenExpiry,
	}).Info("Feishu access token updated")

	return c.accessToken, nil
}

// SendMessage 发送消息
// 支持多种消息类型和格式
func (c *Client) SendMessage(ctx context.Context, receiveID string, msgType MessageType, content string) (*SendMessageResponse, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("feishu integration is not enabled")
	}

	// 获取访问令牌
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// 构建请求
	msgRequest := SendMessageRequest{
		ReceiveID:   receiveID,
		MessageType: msgType,
		Content:     content,
		UUID:        fmt.Sprintf("daily-brief-%d", time.Now().UnixNano()),
	}

	jsonData, err := json.Marshal(msgRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", 
		"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id", 
		bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create message request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	var msgResp SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return nil, fmt.Errorf("failed to decode message response: %w", err)
	}

	// 记录发送日志
	log := &storage.IntegrationLog{
		Platform:    "feishu",
		Action:      "send_message",
		ReceiveID:   receiveID,
		MessageType: string(msgType),
		Status:      "success",
		Response:    fmt.Sprintf("code: %d, msg: %s", msgResp.Code, msgResp.Msg),
	}

	if msgResp.Code != 0 {
		log.Status = "failed"
		log.Error = msgResp.Msg
		if err := c.database.CreateIntegrationLog(log); err != nil {
			c.logger.WithError(err).Error("Failed to save integration log")
		}
		return nil, fmt.Errorf("failed to send message: code=%d, msg=%s", msgResp.Code, msgResp.Msg)
	} else {
		log.MessageID = msgResp.Data.MessageID
	}

	if err := c.database.CreateIntegrationLog(log); err != nil {
		c.logger.WithError(err).Error("Failed to save integration log")
	}

	c.logger.WithFields(logrus.Fields{
		"receive_id":  receiveID,
		"message_id":  msgResp.Data.MessageID,
		"msg_type":    msgType,
		"content_len": len(content),
	}).Info("Message sent to Feishu successfully")

	return &msgResp, nil
}

// SendWebhookMessage 通过Webhook发送消息
// 适用于群机器人场景，支持签名验证
func (c *Client) SendWebhookMessage(ctx context.Context, content interface{}) error {
	if c.config.WebhookURL == "" {
		return fmt.Errorf("webhook URL is not configured")
	}

	// 如果配置了签名密钥，在消息体中添加签名信息
	if c.config.WebhookSecret != "" {
		timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
		signature := c.generateSignature(timestamp, "")
		
		// 创建包含签名的消息体
		signedContent := map[string]interface{}{
			"timestamp": timestamp,
			"sign":      signature,
		}
		
		// 将原始内容合并到签名消息体中
		if contentMap, ok := content.(map[string]interface{}); ok {
			for k, v := range contentMap {
				signedContent[k] = v
			}
		} else {
			return fmt.Errorf("content must be a map[string]interface{} when using signature")
		}
		
		content = signedContent
		
		c.logger.WithFields(logrus.Fields{
			"timestamp": timestamp,
			"signature": signature[:10] + "...", // 只显示前10位用于调试
		}).Debug("Added webhook signature to message body")
	}

	jsonData, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook message: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read webhook response: %w", err)
	}

	// 记录发送日志
	log := &storage.IntegrationLog{
		Platform:    "feishu",
		Action:      "send_webhook",
		MessageType: "webhook",
		Status:      "success",
		Response:    string(respBody),
	}

	if resp.StatusCode >= 400 {
		log.Status = "failed"
		log.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))
		if err := c.database.CreateIntegrationLog(log); err != nil {
			c.logger.WithError(err).Error("Failed to save integration log")
		}
		return fmt.Errorf("webhook request failed: HTTP %d", resp.StatusCode)
	}

	if err := c.database.CreateIntegrationLog(log); err != nil {
		c.logger.WithError(err).Error("Failed to save integration log")
	}

	c.logger.WithFields(logrus.Fields{
		"webhook_url": c.maskURL(c.config.WebhookURL),
		"status_code": resp.StatusCode,
		"content_len": len(jsonData),
	}).Info("Webhook message sent successfully")

	return nil
}

// TestConnection 测试飞书连接
func (c *Client) TestConnection(ctx context.Context) error {
	if !c.IsEnabled() {
		return fmt.Errorf("feishu integration is not enabled")
	}

	// 尝试获取访问令牌
	_, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Feishu: %w", err)
	}

	c.logger.Info("Feishu connection test successful")
	return nil
}

// GetIntegrationStats 获取集成统计信息
func (c *Client) GetIntegrationStats() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled":           c.IsEnabled(),
		"has_app_id":       c.config.AppID != "",
		"has_app_secret":   c.config.AppSecret != "",
		"has_webhook_url":  c.config.WebhookURL != "",
		"has_chat_id":      c.config.ChatID != "",
		"token_valid":      c.accessToken != "" && time.Now().Before(c.tokenExpiry),
		"token_expires_at": c.tokenExpiry,
	}

	return stats
}

// GetConfig 获取飞书配置
func (c *Client) GetConfig() config.FeishuConfig {
	return c.config
}

// maskURL 遮掩URL中的敏感信息
func (c *Client) maskURL(url string) string {
	if len(url) < 20 {
		return "***"
	}
	return url[:10] + "***" + url[len(url)-10:]
}

// validateConfig 验证飞书配置
func (c *Client) validateConfig() error {
	if !c.config.Enabled {
		return fmt.Errorf("feishu integration is disabled")
	}

	if c.config.AppID == "" {
		return fmt.Errorf("feishu app_id is required")
	}

	if c.config.AppSecret == "" {
		return fmt.Errorf("feishu app_secret is required")
	}

	if c.config.WebhookURL == "" && c.config.ChatID == "" {
		return fmt.Errorf("either webhook_url or chat_id must be configured")
	}

	return nil
}

// generateSignature 生成飞书Webhook签名
// 根据飞书官方Java示例：使用 timestamp + "\n" + secret 作为HMAC密钥，对空字节数组计算签名
func (c *Client) generateSignature(timestamp, body string) string {
	// 拼接字符串：timestamp + "\n" + 密钥，作为HMAC密钥
	stringToSign := timestamp + "\n" + c.config.WebhookSecret
	
	// 使用stringToSign作为HMAC密钥，对空字节数组计算签名（按照Java示例）
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte{}) // 对空字节数组计算签名
	signature := h.Sum(nil)
	
	// Base64编码
	return base64.StdEncoding.EncodeToString(signature)
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	// 清理资源
	c.accessToken = ""
	c.tokenExpiry = time.Time{}
	
	c.logger.Info("Feishu client closed")
	return nil
} 