// Package feishu 飞书消息格式化器
// 将日报内容转换为飞书支持的格式
package feishu

import (
	"encoding/json"
	"fmt"
	"strings"

	"daily-brief/internal/report"
	"daily-brief/internal/storage"
)

// MessageFormatter 飞书消息格式化器
// 负责将日报内容转换为飞书支持的消息格式
type MessageFormatter struct {
	// 格式化配置
	maxTextLength    int    // 文本消息最大长度
	useRichText      bool   // 是否使用富文本
	enableEmoji      bool   // 是否启用emoji
	timezone         string // 时区设置
}

// FormatterConfig 格式化器配置
type FormatterConfig struct {
	MaxTextLength int    `json:"max_text_length"` // 最大文本长度
	UseRichText   bool   `json:"use_rich_text"`   // 使用富文本
	EnableEmoji   bool   `json:"enable_emoji"`    // 启用emoji
	Timezone      string `json:"timezone"`        // 时区
}

// FeishuMessage 飞书消息结构
type FeishuMessage struct {
	MsgType string      `json:"msg_type"`
	Content interface{} `json:"content"`
}

// TextContent 文本消息内容
type TextContent struct {
	Text string `json:"text"`
}

// RichTextContent 富文本消息内容
type RichTextContent struct {
	RichText RichTextElements `json:"rich_text"`
}

// RichTextElements 富文本元素
type RichTextElements struct {
	Elements [][]RichTextElement `json:"elements"`
}

// RichTextElement 富文本元素
type RichTextElement struct {
	Tag    string                 `json:"tag"`
	Text   string                 `json:"text,omitempty"`
	Style  map[string]interface{} `json:"style,omitempty"`
	Href   string                 `json:"href,omitempty"`
	UserID string                 `json:"user_id,omitempty"`
}

// CardContent 卡片消息内容
type CardContent struct {
	Card CardElement `json:"card"`
}

// CardElement 卡片元素
type CardElement struct {
	Header   *CardHeader    `json:"header,omitempty"`
	Elements []CardBodyElement `json:"elements"`
}

// CardHeader 卡片头部
type CardHeader struct {
	Title     CardTitle `json:"title"`
	Template  string    `json:"template,omitempty"`
	Icon      *CardIcon `json:"icon,omitempty"`
}

// CardTitle 卡片标题
type CardTitle struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// CardIcon 卡片图标
type CardIcon struct {
	Tag   string `json:"tag"`
	Token string `json:"token"`
}

// CardBodyElement 卡片主体元素
type CardBodyElement struct {
	Tag      string                 `json:"tag"`
	Text     *CardText             `json:"text,omitempty"`
	Fields   []CardField           `json:"fields,omitempty"`
	Elements []CardBodyElement     `json:"elements,omitempty"`
}

// CardText 卡片文本
type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// CardField 卡片字段
type CardField struct {
	IsShort bool      `json:"is_short"`
	Text    CardText  `json:"text"`
}

// NewMessageFormatter 创建新的消息格式化器
func NewMessageFormatter(config FormatterConfig) *MessageFormatter {
	if config.MaxTextLength <= 0 {
		config.MaxTextLength = 8000 // 飞书文本消息限制
	}
	if config.Timezone == "" {
		config.Timezone = "Asia/Shanghai"
	}

	return &MessageFormatter{
		maxTextLength: config.MaxTextLength,
		useRichText:   config.UseRichText,
		enableEmoji:   config.EnableEmoji,
		timezone:      config.Timezone,
	}
}

// FormatReportAsText 将日报格式化为文本消息
func (f *MessageFormatter) FormatReportAsText(data *report.ReportData) (string, error) {
	var builder strings.Builder

	// 标题
	title := "📊 Daily Brief"
	if f.enableEmoji {
		title = "📊 Daily Brief"
	} else {
		title = "Daily Brief"
	}
	
	builder.WriteString(fmt.Sprintf("%s - %s\n", title, data.Date.Format("2006年01月02日")))
	builder.WriteString(strings.Repeat("=", 30) + "\n\n")

	// 核心统计
	builder.WriteString("📈 今日概览\n")
	completionRate := 0.0
	if data.Summary.TotalTasks > 0 {
		completionRate = float64(data.Summary.CompletedTasks) / float64(data.Summary.TotalTasks) * 100
	}
	
	builder.WriteString(fmt.Sprintf("• 完成任务：%d/%d (%.1f%%)\n", 
		data.Summary.CompletedTasks, data.Summary.TotalTasks, completionRate))
	builder.WriteString(fmt.Sprintf("• 代码提交：%d 次\n", data.Summary.TotalCommits))
	builder.WriteString(fmt.Sprintf("• 代码变更：+%d -%d 行\n", 
		data.Summary.LinesAdded, data.Summary.LinesDeleted))
	builder.WriteString(fmt.Sprintf("• 生产力评分：%.1f/100\n\n", data.Summary.ProductivityScore))

	// 重点任务
	if len(data.Summary.HighlightTasks) > 0 {
		builder.WriteString("⭐ 重点任务\n")
		for i, task := range data.Summary.HighlightTasks {
			if i >= 5 { // 限制显示数量
				builder.WriteString(fmt.Sprintf("• ... 还有 %d 个重点任务\n", len(data.Summary.HighlightTasks)-i))
				break
			}
			builder.WriteString(fmt.Sprintf("• %s\n", task))
		}
		builder.WriteString("\n")
	}

	// 主要提交
	if len(data.Commits) > 0 {
		builder.WriteString("💻 代码提交\n")
		for i, commit := range data.Commits {
			if i >= 5 { // 限制显示数量
				builder.WriteString(fmt.Sprintf("• ... 还有 %d 个提交\n", len(data.Commits)-i))
				break
			}
			icon := f.getCommitIcon(commit.Type)
			builder.WriteString(fmt.Sprintf("%s %s (%s)\n", 
				icon, commit.Message, commit.Date.Format("15:04")))
		}
		builder.WriteString("\n")
	}

	// 生成信息
	builder.WriteString(fmt.Sprintf("📝 %s 自动生成", data.Metadata.GeneratedAt.Format("15:04")))

	content := builder.String()
	
	// 检查长度限制
	if len(content) > f.maxTextLength {
		content = content[:f.maxTextLength-50] + "\n...\n内容过长，已截断。请查看完整版本。"
	}

	return content, nil
}

// FormatReportAsCard 将日报格式化为卡片消息
func (f *MessageFormatter) FormatReportAsCard(data *report.ReportData) (*CardContent, error) {
	card := &CardContent{
		Card: CardElement{
			Header: &CardHeader{
				Title: CardTitle{
					Tag:     "text",
					Content: fmt.Sprintf("📊 Daily Brief - %s", data.Date.Format("01月02日")),
				},
				Template: "blue",
				Icon: &CardIcon{
					Tag:   "standard_icon",
					Token: "chart_line_outlined",
				},
			},
			Elements: []CardBodyElement{},
		},
	}

	// 概览统计
	card.Card.Elements = append(card.Card.Elements, CardBodyElement{
		Tag: "div",
		Fields: []CardField{
			{
				IsShort: true,
				Text: CardText{
					Tag:     "lark_md",
					Content: fmt.Sprintf("**完成任务**\n%d/%d (%.1f%%)", 
						data.Summary.CompletedTasks, 
						data.Summary.TotalTasks,
						float64(data.Summary.CompletedTasks)/float64(data.Summary.TotalTasks)*100),
				},
			},
			{
				IsShort: true,
				Text: CardText{
					Tag:     "lark_md",
					Content: fmt.Sprintf("**代码提交**\n%d 次", data.Summary.TotalCommits),
				},
			},
			{
				IsShort: true,
				Text: CardText{
					Tag:     "lark_md",
					Content: fmt.Sprintf("**代码变更**\n+%d -%d 行", 
						data.Summary.LinesAdded, data.Summary.LinesDeleted),
				},
			},
			{
				IsShort: true,
				Text: CardText{
					Tag:     "lark_md",
					Content: fmt.Sprintf("**生产力评分**\n%.1f/100", data.Summary.ProductivityScore),
				},
			},
		},
	})

	// 分隔线
	card.Card.Elements = append(card.Card.Elements, CardBodyElement{
		Tag: "hr",
	})

	// 重点任务
	if len(data.Summary.HighlightTasks) > 0 {
		taskContent := "**⭐ 重点任务**\n"
		for i, task := range data.Summary.HighlightTasks {
			if i >= 3 { // 限制显示数量
				taskContent += fmt.Sprintf("... 还有 %d 个任务\n", len(data.Summary.HighlightTasks)-i)
				break
			}
			taskContent += fmt.Sprintf("• %s\n", task)
		}

		card.Card.Elements = append(card.Card.Elements, CardBodyElement{
			Tag: "div",
			Text: &CardText{
				Tag:     "lark_md",
				Content: taskContent,
			},
		})
	}

	// 代码提交
	if len(data.Commits) > 0 {
		commitContent := "**💻 代码提交**\n"
		for i, commit := range data.Commits {
			if i >= 3 { // 限制显示数量
				commitContent += fmt.Sprintf("... 还有 %d 个提交\n", len(data.Commits)-i)
				break
			}
			icon := f.getCommitIcon(commit.Type)
			commitContent += fmt.Sprintf("%s %s (%s)\n", 
				icon, commit.Message, commit.Date.Format("15:04"))
		}

		card.Card.Elements = append(card.Card.Elements, CardBodyElement{
			Tag: "div",
			Text: &CardText{
				Tag:     "lark_md",
				Content: commitContent,
			},
		})
	}

	// 页脚
	card.Card.Elements = append(card.Card.Elements, CardBodyElement{
		Tag: "note",
		Text: &CardText{
			Tag:     "lark_md",
			Content: fmt.Sprintf("📝 %s 自动生成", data.Metadata.GeneratedAt.Format("2006-01-02 15:04")),
		},
	})

	return card, nil
}

// FormatReportAsWebhook 将日报格式化为Webhook消息
func (f *MessageFormatter) FormatReportAsWebhook(data *report.ReportData) (map[string]interface{}, error) {
	// 使用富文本格式
	content, err := f.FormatReportAsText(data)
	if err != nil {
		return nil, err
	}

	webhook := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]interface{}{
			"text": content,
		},
	}

	return webhook, nil
}

// FormatDailyReport 格式化已保存的日报
func (f *MessageFormatter) FormatDailyReport(report *storage.DailyReport, format MessageType) (interface{}, error) {
	switch format {
	case MessageTypeText:
		// 对于已保存的日报，我们使用简化的文本格式
		content := fmt.Sprintf("📊 %s\n\n", report.Title)
		
		// 截取内容前500字符作为预览
		preview := report.Content
		if len(preview) > 500 {
			preview = preview[:500] + "...\n\n点击查看完整日报"
		}
		
		content += preview
		content += fmt.Sprintf("\n📝 生成于 %s", report.CreatedAt.Format("2006-01-02 15:04"))
		
		return content, nil
		
	case MessageTypeCard:
		// 创建卡片消息
		card := &CardContent{
			Card: CardElement{
				Header: &CardHeader{
					Title: CardTitle{
						Tag:     "text",
						Content: "📊 " + report.Title,
					},
					Template: "blue",
				},
				Elements: []CardBodyElement{
					{
						Tag: "div",
						Text: &CardText{
							Tag:     "lark_md",
							Content: "**状态**: " + report.Status,
						},
					},
					{
						Tag: "div",
						Text: &CardText{
							Tag:     "lark_md",
							Content: "**模板**: " + report.Template,
						},
					},
					{
						Tag: "div",
						Text: &CardText{
							Tag:     "lark_md",
							Content: fmt.Sprintf("**生成时间**: %s", report.CreatedAt.Format("2006-01-02 15:04")),
						},
					},
				},
			},
		}
		
		return card, nil
		
	default:
		return f.FormatDailyReport(report, MessageTypeText)
	}
}

// getCommitIcon 获取提交类型图标
func (f *MessageFormatter) getCommitIcon(commitType string) string {
	if !f.enableEmoji {
		return "•"
	}

	switch commitType {
	case "feature":
		return "✨"
	case "bugfix":
		return "🐛"
	case "docs":
		return "📚"
	case "style":
		return "💄"
	case "refactor":
		return "♻️"
	case "test":
		return "🧪"
	case "chore":
		return "🔧"
	case "perf":
		return "⚡"
	default:
		return "📝"
	}
}

// CreateTextMessage 创建文本消息
func (f *MessageFormatter) CreateTextMessage(text string) *FeishuMessage {
	return &FeishuMessage{
		MsgType: "text",
		Content: TextContent{
			Text: text,
		},
	}
}

// CreateCardMessage 创建卡片消息
func (f *MessageFormatter) CreateCardMessage(card *CardContent) *FeishuMessage {
	return &FeishuMessage{
		MsgType: "interactive",
		Content: card.Card,
	}
}

// ToJSON 转换为JSON字符串
func (f *MessageFormatter) ToJSON(message interface{}) (string, error) {
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}
	return string(jsonBytes), nil
}

// ValidateMessage 验证消息格式
func (f *MessageFormatter) ValidateMessage(message *FeishuMessage) error {
	if message.MsgType == "" {
		return fmt.Errorf("message type is required")
	}

	if message.Content == nil {
		return fmt.Errorf("message content is required")
	}

	switch message.MsgType {
	case "text":
		if content, ok := message.Content.(TextContent); ok {
			if len(content.Text) > f.maxTextLength {
				return fmt.Errorf("text content exceeds maximum length (%d)", f.maxTextLength)
			}
		}
	case "interactive":
		// 卡片消息验证可以在这里添加
	}

	return nil
} 