// Package workpoint 提供工作要点管理功能
// 包括分类管理、智能标签、优先级管理、模板系统等
package workpoint

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"daily-brief/internal/config"
	"daily-brief/internal/storage"

	"github.com/sirupsen/logrus"
)

// Manager 工作要点管理器
// 负责工作要点的创建、分类、标签管理和优先级评估
type Manager struct {
	database storage.Database    // 数据库接口
	config   config.ReportConfig // 报告配置
	logger   *logrus.Logger      // 日志器
	
	// 内置分类器和标签系统
	categoryRules   []CategoryRule   // 分类规则
	tagRules        []TagRule        // 标签规则
	priorityRules   []PriorityRule   // 优先级规则
}

// CategoryRule 分类规则
// 用于自动将工作要点分类到合适的类别
type CategoryRule struct {
	Name        string   `json:"name"`        // 规则名称
	Category    string   `json:"category"`    // 目标类别
	Keywords    []string `json:"keywords"`    // 关键词列表
	Patterns    []string `json:"patterns"`    // 正则表达式模式
	Weight      int      `json:"weight"`      // 规则权重
	Description string   `json:"description"` // 规则描述
}

// TagRule 标签规则
// 用于自动为工作要点添加标签
type TagRule struct {
	Name        string   `json:"name"`        // 规则名称
	Tag         string   `json:"tag"`         // 标签名称
	Keywords    []string `json:"keywords"`    // 关键词列表
	Patterns    []string `json:"patterns"`    // 正则表达式模式
	Weight      int      `json:"weight"`      // 规则权重
	Color       string   `json:"color"`       // 标签颜色
	Description string   `json:"description"` // 规则描述
}

// PriorityRule 优先级规则
// 用于自动评估工作要点的优先级
type PriorityRule struct {
	Name        string   `json:"name"`        // 规则名称
	Priority    string   `json:"priority"`    // 优先级：high/medium/low
	Keywords    []string `json:"keywords"`    // 关键词列表
	Patterns    []string `json:"patterns"`    // 正则表达式模式
	Weight      int      `json:"weight"`      // 规则权重
	Score       int      `json:"score"`       // 优先级分数
	Description string   `json:"description"` // 规则描述
}

// WorkPointAnalysis 工作要点分析结果
type WorkPointAnalysis struct {
	OriginalPoint   *storage.WorkPoint     `json:"original_point"`   // 原始要点
	SuggestedTags   []string               `json:"suggested_tags"`   // 建议标签
	SuggestedPriority string               `json:"suggested_priority"` // 建议优先级
	CategoryScore   map[string]int         `json:"category_score"`   // 分类评分
	TagScore        map[string]int         `json:"tag_score"`        // 标签评分
	PriorityScore   int                    `json:"priority_score"`   // 优先级评分
	Confidence      float64                `json:"confidence"`       // 置信度
}

// NewManager 创建新的工作要点管理器
func NewManager(db storage.Database, cfg config.ReportConfig, logger *logrus.Logger) *Manager {
	manager := &Manager{
		database: db,
		config:   cfg,
		logger:   logger,
	}
	
	// 初始化默认规则
	manager.initializeDefaultRules()
	
	return manager
}

// initializeDefaultRules 初始化默认的分类、标签和优先级规则
func (m *Manager) initializeDefaultRules() {
	// 默认分类规则
	m.categoryRules = []CategoryRule{
		{
			Name:        "开发工作",
			Category:    "development",
			Keywords:    []string{"开发", "编码", "代码", "功能", "bug", "修复", "实现", "代码审查"},
			Patterns:    []string{`(?i)(fix|feat|add|implement|develop|code|debug)`},
			Weight:      10,
			Description: "软件开发相关的工作",
		},
		{
			Name:        "测试工作",
			Category:    "testing",
			Keywords:    []string{"测试", "单元测试", "集成测试", "自动化测试", "测试用例", "质量保证"},
			Patterns:    []string{`(?i)(test|qa|quality|unit.*test|integration.*test)`},
			Weight:      8,
			Description: "软件测试相关的工作",
		},
		{
			Name:        "文档工作",
			Category:    "documentation",
			Keywords:    []string{"文档", "说明", "手册", "注释", "README", "API文档", "用户指南"},
			Patterns:    []string{`(?i)(doc|documentation|readme|manual|guide|comment)`},
			Weight:      6,
			Description: "文档编写和维护工作",
		},
		{
			Name:        "会议沟通",
			Category:    "meeting",
			Keywords:    []string{"会议", "讨论", "沟通", "汇报", "评审", "站会", "需求分析"},
			Patterns:    []string{`(?i)(meeting|discuss|communicate|standup|review|sync)`},
			Weight:      7,
			Description: "会议和沟通相关的工作",
		},
		{
			Name:        "运维部署",
			Category:    "devops",
			Keywords:    []string{"部署", "运维", "监控", "配置", "环境", "服务器", "Docker", "CI/CD"},
			Patterns:    []string{`(?i)(deploy|devops|monitor|config|server|docker|ci|cd)`},
			Weight:      9,
			Description: "运维和部署相关的工作",
		},
		{
			Name:        "学习研究",
			Category:    "learning",
			Keywords:    []string{"学习", "研究", "调研", "技术预研", "培训", "分享", "技术方案"},
			Patterns:    []string{`(?i)(learn|study|research|training|investigate|explore)`},
			Weight:      5,
			Description: "学习和研究相关的工作",
		},
	}
	
	// 默认标签规则
	m.tagRules = []TagRule{
		{
			Name:        "紧急任务",
			Tag:         "urgent",
			Keywords:    []string{"紧急", "urgent", "线上问题", "故障", "bug", "hotfix", "critical"},
			Patterns:    []string{`(?i)(urgent|critical|emergency|hotfix|p0|高优先级)`},
			Weight:      15,
			Color:       "red",
			Description: "需要立即处理的紧急任务",
		},
		{
			Name:        "新功能",
			Tag:         "feature",
			Keywords:    []string{"新功能", "feature", "需求", "开发", "实现", "添加"},
			Patterns:    []string{`(?i)(feature|new.*feature|implement|add.*feature)`},
			Weight:      8,
			Color:       "blue",
			Description: "新功能开发相关的工作",
		},
		{
			Name:        "性能优化",
			Tag:         "performance",
			Keywords:    []string{"优化", "性能", "performance", "慢查询", "性能调优", "内存优化"},
			Patterns:    []string{`(?i)(optimize|performance|tuning|slow.*query|memory.*optimization)`},
			Weight:      7,
			Color:       "green",
			Description: "性能优化相关的工作",
		},
		{
			Name:        "重构代码",
			Tag:         "refactor",
			Keywords:    []string{"重构", "refactor", "代码优化", "架构调整", "重构代码"},
			Patterns:    []string{`(?i)(refactor|refactoring|restructure|code.*optimization)`},
			Weight:      6,
			Color:       "yellow",
			Description: "代码重构相关的工作",
		},
		{
			Name:        "安全相关",
			Tag:         "security",
			Keywords:    []string{"安全", "security", "权限", "认证", "授权", "漏洞", "安全审计"},
			Patterns:    []string{`(?i)(security|auth|permission|vulnerability|audit)`},
			Weight:      12,
			Color:       "orange",
			Description: "安全相关的工作",
		},
		{
			Name:        "技术债务",
			Tag:         "tech-debt",
			Keywords:    []string{"技术债务", "tech debt", "遗留问题", "历史问题", "代码清理"},
			Patterns:    []string{`(?i)(tech.*debt|legacy|deprecated|cleanup|outdated)`},
			Weight:      4,
			Color:       "gray",
			Description: "技术债务和遗留问题",
		},
	}
	
	// 默认优先级规则
	m.priorityRules = []PriorityRule{
		{
			Name:        "线上故障",
			Priority:    "high",
			Keywords:    []string{"线上", "故障", "bug", "紧急", "崩溃", "异常", "错误"},
			Patterns:    []string{`(?i)(production.*bug|critical.*bug|crash|error|exception|failure)`},
			Weight:      20,
			Score:       100,
			Description: "影响线上服务的问题",
		},
		{
			Name:        "截止时间临近",
			Priority:    "high",
			Keywords:    []string{"deadline", "截止", "今天", "明天", "urgent", "紧急"},
			Patterns:    []string{`(?i)(deadline|due.*today|due.*tomorrow|urgent|asap)`},
			Weight:      15,
			Score:       90,
			Description: "截止时间临近的任务",
		},
		{
			Name:        "重要功能",
			Priority:    "medium",
			Keywords:    []string{"重要", "核心", "关键", "主要", "优先"},
			Patterns:    []string{`(?i)(important|critical|key|main|priority|core.*feature)`},
			Weight:      10,
			Score:       70,
			Description: "重要的功能和任务",
		},
		{
			Name:        "优化改进",
			Priority:    "medium",
			Keywords:    []string{"优化", "改进", "提升", "增强", "完善"},
			Patterns:    []string{`(?i)(optimize|improve|enhance|upgrade|refine)`},
			Weight:      8,
			Score:       60,
			Description: "优化和改进类的工作",
		},
		{
			Name:        "文档和学习",
			Priority:    "low",
			Keywords:    []string{"文档", "学习", "研究", "调研", "培训", "分享"},
			Patterns:    []string{`(?i)(document|learn|study|research|training|share)`},
			Weight:      5,
			Score:       30,
			Description: "文档和学习相关的工作",
		},
		{
			Name:        "日常维护",
			Priority:    "low",
			Keywords:    []string{"维护", "清理", "整理", "日常", "例行"},
			Patterns:    []string{`(?i)(maintain|cleanup|organize|routine|regular)`},
			Weight:      3,
			Score:       20,
			Description: "日常维护类的工作",
		},
	}
	
	m.logger.Info("Initialized work point management rules")
	m.logger.Infof("Loaded %d category rules, %d tag rules, %d priority rules", 
		len(m.categoryRules), len(m.tagRules), len(m.priorityRules))
}

// CreateWorkPoint 创建工作要点（带智能分析）
func (m *Manager) CreateWorkPoint(title, description, category string) (*storage.WorkPoint, error) {
	// 创建基础工作要点
	point := &storage.WorkPoint{
		Title:       title,
		Description: description,
		Category:    category,
		Date:        time.Now(),
		Priority:    1, // 默认优先级
		Tags:        "", // 将通过分析获得
		Completed:   false,
	}
	
	// 执行智能分析
	analysis := m.AnalyzeWorkPoint(point)
	
	// 应用分析结果
	if analysis.SuggestedPriority != "" {
		point.Priority = m.priorityStringToInt(analysis.SuggestedPriority)
	}
	
	if len(analysis.SuggestedTags) > 0 {
		point.Tags = strings.Join(analysis.SuggestedTags, ",")
	}
	
	// 如果没有指定分类，使用分析结果
	if point.Category == "" {
		point.Category = m.getBestCategory(analysis.CategoryScore)
	}
	
	// 存储到数据库
	if err := m.database.CreateWorkPoint(point); err != nil {
		return nil, fmt.Errorf("failed to create work point: %w", err)
	}
	
	m.logger.WithFields(logrus.Fields{
		"title":    point.Title,
		"category": point.Category,
		"priority": point.Priority,
		"tags":     point.Tags,
	}).Info("Created work point with analysis")
	
	return point, nil
}

// AnalyzeWorkPoint 分析工作要点
func (m *Manager) AnalyzeWorkPoint(point *storage.WorkPoint) *WorkPointAnalysis {
	analysis := &WorkPointAnalysis{
		OriginalPoint: point,
		CategoryScore: make(map[string]int),
		TagScore:      make(map[string]int),
	}
	
	// 组合分析文本（标题 + 描述）
	text := strings.ToLower(point.Title + " " + point.Description)
	
	// 分类分析
	for _, rule := range m.categoryRules {
		score := m.calculateRuleScore(text, rule.Keywords, rule.Patterns, rule.Weight)
		if score > 0 {
			analysis.CategoryScore[rule.Category] += score
		}
	}
	
	// 标签分析
	for _, rule := range m.tagRules {
		score := m.calculateRuleScore(text, rule.Keywords, rule.Patterns, rule.Weight)
		if score > 0 {
			analysis.TagScore[rule.Tag] += score
		}
	}
	
	// 优先级分析
	maxPriorityScore := 0
	var suggestedPriority string
	
	for _, rule := range m.priorityRules {
		score := m.calculateRuleScore(text, rule.Keywords, rule.Patterns, rule.Weight)
		if score > 0 && rule.Score > maxPriorityScore {
			maxPriorityScore = rule.Score
			suggestedPriority = rule.Priority
		}
	}
	
	analysis.PriorityScore = maxPriorityScore
	analysis.SuggestedPriority = suggestedPriority
	
	// 提取建议的标签（分数最高的前3个）
	analysis.SuggestedTags = m.getTopTags(analysis.TagScore, 3)
	
	// 计算整体置信度
	analysis.Confidence = m.calculateConfidence(analysis)
	
	return analysis
}

// calculateRuleScore 计算规则评分
func (m *Manager) calculateRuleScore(text string, keywords []string, patterns []string, weight int) int {
	score := 0
	
	// 关键词匹配
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			score += weight
		}
	}
	
	// 正则表达式匹配
	for _, pattern := range patterns {
		if matched, err := regexp.MatchString(pattern, text); err == nil && matched {
			score += weight * 2 // 正则匹配权重更高
		}
	}
	
	return score
}

// getBestCategory 获取最佳分类
func (m *Manager) getBestCategory(categoryScore map[string]int) string {
	if len(categoryScore) == 0 {
		return "other"
	}
	
	maxScore := 0
	bestCategory := "other"
	
	for category, score := range categoryScore {
		if score > maxScore {
			maxScore = score
			bestCategory = category
		}
	}
	
	return bestCategory
}

// getTopTags 获取评分最高的标签
func (m *Manager) getTopTags(tagScore map[string]int, limit int) []string {
	type TagScore struct {
		Tag   string
		Score int
	}
	
	var tagScores []TagScore
	for tag, score := range tagScore {
		if score > 0 {
			tagScores = append(tagScores, TagScore{Tag: tag, Score: score})
		}
	}
	
	// 按分数排序
	sort.Slice(tagScores, func(i, j int) bool {
		return tagScores[i].Score > tagScores[j].Score
	})
	
	// 提取前N个标签
	var tags []string
	for i := 0; i < len(tagScores) && i < limit; i++ {
		tags = append(tags, tagScores[i].Tag)
	}
	
	return tags
}

// calculateConfidence 计算置信度
func (m *Manager) calculateConfidence(analysis *WorkPointAnalysis) float64 {
	totalScore := 0
	maxScore := 0
	
	// 计算分类置信度
	for _, score := range analysis.CategoryScore {
		totalScore += score
		if score > maxScore {
			maxScore = score
		}
	}
	
	// 计算标签置信度
	for _, score := range analysis.TagScore {
		totalScore += score
	}
	
	// 加入优先级分数
	totalScore += analysis.PriorityScore
	
	if totalScore == 0 {
		return 0.0
	}
	
	// 简单的置信度计算：最高分数 / 总分数
	confidence := float64(maxScore) / float64(totalScore)
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// priorityStringToInt 将优先级字符串转换为整数
func (m *Manager) priorityStringToInt(priority string) int {
	switch strings.ToLower(priority) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 1
	}
}

// GetWorkPointsByCategory 按分类获取工作要点
func (m *Manager) GetWorkPointsByCategory(category string, date time.Time) ([]storage.WorkPoint, error) {
	return m.database.GetWorkPointsByCategory(category, date)
}

// GetWorkPointsWithAnalysis 获取工作要点并包含分析信息
func (m *Manager) GetWorkPointsWithAnalysis(date time.Time) ([]*WorkPointAnalysis, error) {
	points, err := m.database.GetWorkPoints(date)
	if err != nil {
		return nil, err
	}
	
	var analyses []*WorkPointAnalysis
	for _, point := range points {
		analysis := m.AnalyzeWorkPoint(&point)
		analyses = append(analyses, analysis)
	}
	
	return analyses, nil
}

// GetCategoryStats 获取分类统计信息
func (m *Manager) GetCategoryStats(startDate, endDate time.Time) (map[string]int, error) {
	commits, err := m.database.GetCommitsByDateRange(startDate, endDate)
	if err != nil {
		return nil, err
	}
	
	stats := make(map[string]int)
	for _, commit := range commits {
		if commit.Type != "" {
			stats[commit.Type]++
		}
	}
	
	return stats, nil
}

// UpdateWorkPointTags 更新工作要点的标签
func (m *Manager) UpdateWorkPointTags(pointID uint, tags []string) error {
	point, err := m.database.GetWorkPoints(time.Now())
	if err != nil {
		return err
	}
	
	// 找到对应的工作要点
	for _, p := range point {
		if p.ID == pointID {
			p.Tags = strings.Join(tags, ",")
			return m.database.UpdateWorkPoint(&p)
		}
	}
	
	return fmt.Errorf("work point not found: %d", pointID)
} 