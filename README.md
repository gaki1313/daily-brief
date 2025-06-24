# 日报生成器 (Daily Brief Generator)

一个基于Go语言开发的智能日报生成器，支持AI增强的内容生成。能够自动收集GitLab提交记录和用户要点，使用AI生成专业的工作日报，并通过飞书机器人自动推送。

## ✨ 最新功能

- 🤖 **AI智能生成**：集成Ollama/OpenAI，AI自动生成自然流畅的工作日报
- 🔗 **GitLab深度集成**：支持GitLab API v3，智能获取用户提交记录  
- 📱 **飞书自动推送**：支持Webhook和API两种方式推送到飞书群聊
- 🎯 **用户要点融合**：自定义工作成果与代码提交智能融合
- 📊 **多数据源支持**：本地Git + GitLab + 工作要点的综合数据收集

## 🚀 快速体验

### 1. AI日报生成API
```bash
# 生成AI增强的GitLab日报并推送到飞书
curl -X POST http://localhost:3000/api/v1/integrations/feishu/send/ai-gitlab-report \
  -H "Content-Type: application/json" \
  -d '{
    "date": "2025-06-23",
    "work_items": ["1. 七月绩效制定完成", "2. 代码重构优化"]
  }'
```

### 2. 飞书推送功能
```bash
# 标准GitLab日报推送（混合数据源）
curl -X POST http://localhost:3000/api/v1/integrations/feishu/send/report \
  -H "Content-Type: application/json" \
  -d '{"date": "2025-06-23", "template": "standard"}'

# 纯GitLab数据日报推送
curl -X POST http://localhost:3000/api/v1/integrations/feishu/send/gitlab-report \
  -H "Content-Type: application/json" \
  -d '{"date": "2025-06-23", "template": "detailed"}'
```

### 3. GitLab集成状态检查
```bash
# 检查GitLab连接状态
curl http://localhost:3000/api/v1/integrations/gitlab/status

# 获取GitLab用户信息
curl http://localhost:3000/api/v1/integrations/gitlab/user
```

## 🎯 项目目标

- **AI智能日报生成**：基于AI技术生成自然流畅的工作日报
- **GitLab深度集成**：无缝对接GitLab获取真实提交数据
- **多平台自动推送**：支持飞书等第三方平台自动推送
- **模块化微服务架构**：易于扩展和维护的分布式设计
- **学习导向**：通过完备的代码注释帮助Go语言学习

## 🏗️ 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   数据收集层     │    │   AI生成层      │    │   推送集成层     │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ • GitLab API    │    │ • Ollama集成    │    │ • 飞书Webhook   │
│ • 本地Git扫描   │    │ • OpenAI集成    │    │ • 飞书API       │
│ • 工作要点输入  │    │ • 智能提示词    │    │ • 邮件推送       │
│ • 配置管理      │    │ • 内容优化      │    │ • 多平台扩展     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   存储管理层     │
                    ├─────────────────┤
                    │ • SQLite数据库  │
                    │ • 缓存管理      │
                    │ • 日志管理      │
                    └─────────────────┘
```

## 📋 功能模块

### 1. 核心功能模块

#### 1.1 GitLab集成器 (GitLab Integration)
- **功能**：深度集成GitLab API获取提交记录
- **特性**：
  - 支持GitLab API v3协议
  - 智能用户身份匹配（邮箱/用户名）
  - 超优化的分支搜索策略
  - 按日期范围精确筛选提交
  - 提取详细提交信息（作者、时间、消息、文件变更）

#### 1.2 AI日报生成器 (AI Report Generator)
- **功能**：使用AI技术生成自然流畅的工作日报
- **特性**：
  - 集成Ollama本地大模型
  - 支持OpenAI API
  - 智能提示词工程
  - 用户要点与代码提交融合
  - 专业语言风格控制

#### 1.3 工作要点管理器 (Work Point Manager)
- **功能**：管理用户手动输入的工作要点
- **特性**：
  - 分类管理（完成项、进行中、计划项）
  - 优先级设置
  - 智能分析与标签
  - 历史记录追踪

#### 1.4 飞书推送集成 (Feishu Integration)
- **功能**：自动推送日报到飞书平台
- **特性**：
  - 支持Webhook机器人推送
  - 支持飞书开放平台API
  - 多种消息格式（文本、卡片、Markdown）
  - 自动签名验证
  - 错误重试机制

### 2. 辅助功能模块

#### 2.1 配置管理 (Config Manager)
- 应用配置管理
- 用户偏好设置
- 第三方服务配置

#### 2.2 数据存储 (Data Storage)
- SQLite数据库
- 缓存管理
- 备份恢复

#### 2.3 日志系统 (Logging System)
- 结构化日志
- 日志轮转
- 错误追踪

## 🛠️ 技术栈

### 后端技术
- **语言**：Go 1.21+
- **Web框架**：Gin (轻量级HTTP框架)
- **数据库**：SQLite (嵌入式数据库)
- **ORM**：GORM (Go对象关系映射)
- **配置管理**：Viper (配置文件处理)
- **日志**：Logrus (结构化日志)
- **Git操作**：go-git + GitLab API Client
- **AI集成**：Ollama + OpenAI API
- **HTTP客户端**：resty (HTTP请求库)

### AI & 第三方集成
- **AI模型**：Ollama（本地部署）、OpenAI GPT系列
- **GitLab API**：v3协议，支持自建GitLab实例
- **飞书开放平台**：Webhook + API双模式
- **消息推送**：支持多种消息格式和平台扩展

### 开发工具
- **构建工具**：Go Modules
- **测试框架**：Go原生testing + testify
- **代码质量**：golangci-lint
- **文档生成**：godoc

## 📁 项目结构

```
daily-brief/
├── cmd/                    # 命令行入口
│   └── main.go            # 主程序入口
├── internal/              # 内部包（不对外暴露）
│   ├── ai/               # AI集成
│   │   └── client.go     # AI客户端（Ollama/OpenAI）
│   ├── config/           # 配置管理
│   │   ├── config.go     # 配置结构定义
│   │   └── loader.go     # 配置加载器
│   ├── feishu/           # 飞书集成
│   │   ├── client.go     # 飞书客户端
│   │   ├── formatter.go  # 消息格式化
│   │   └── scheduler.go  # 定时推送
│   ├── git/              # Git操作
│   │   ├── scanner.go    # 本地Git扫描
│   │   └── scheduler.go  # Git调度器
│   ├── gitlab/           # GitLab集成
│   │   └── client.go     # GitLab API客户端
│   ├── report/           # 日报生成
│   │   ├── generator.go  # 日报生成引擎
│   │   └── templates.go  # 模板系统
│   ├── server/           # HTTP服务器
│   │   ├── handlers.go   # 路由处理器
│   │   ├── middleware.go # 中间件
│   │   ├── routes.go     # 路由定义
│   │   └── server.go     # 服务器主体
│   ├── storage/          # 数据存储
│   │   ├── database.go   # 数据库操作
│   │   ├── migration.go  # 数据库迁移
│   │   └── models.go     # 数据模型
│   └── workpoint/        # 工作要点管理
│       └── manager.go    # 要点管理器
├── pkg/                   # 公共包（可对外暴露）
│   ├── logger/           # 日志工具
│   ├── utils/            # 通用工具
│   └── errors/           # 错误处理
├── configs/              # 配置文件
│   ├── config.yaml       # 主配置文件
│   └── templates/        # 日报模板
├── web/                  # Web界面（可选）
│   ├── static/           # 静态资源
│   └── templates/        # HTML模板
├── scripts/              # 构建和部署脚本
├── docs/                 # 文档
├── test/                 # 测试文件
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 🚀 实现计划

### 第一阶段：基础框架搭建 (1-2周)
1. **项目初始化**
   - Go模块初始化
   - 目录结构创建
   - 基础依赖配置

2. **核心架构**
   - 配置管理系统
   - 日志系统
   - 数据库模型设计

3. **Git数据收集器**
   - 基础Git操作封装
   - 提交记录解析
   - 数据存储

### 第二阶段：核心功能实现 (2-3周)
1. **要点管理系统**
   - 数据模型设计
   - CRUD操作
   - 分类和标签功能

2. **日报生成引擎**
   - 模板系统设计
   - 内容聚合算法
   - 格式化输出

3. **基础Web界面**
   - REST API设计
   - 简单Web界面

### 第三阶段：集成和优化 (1-2周)
1. **飞书机器人集成**
   - 飞书API封装
   - 消息推送功能
   - 错误处理机制

2. **功能完善**
   - 配置界面
   - 批量操作
   - 数据导入导出

### 第四阶段：测试和部署 (1周)
1. **测试覆盖**
   - 单元测试
   - 集成测试
   - 性能测试

2. **部署优化**
   - Docker容器化
   - 部署脚本
   - 监控告警

## 📊 数据模型设计

### 核心实体

#### 1. Repository (仓库)
```go
type Repository struct {
    ID          uint      `gorm:"primaryKey"`
    Name        string    `gorm:"not null"`
    Path        string    `gorm:"not null"`
    URL         string    
    Branch      string    `gorm:"default:main"`
    Enabled     bool      `gorm:"default:true"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### 2. Commit (提交记录)
```go
type Commit struct {
    ID           uint      `gorm:"primaryKey"`
    Hash         string    `gorm:"uniqueIndex;not null"`
    RepositoryID uint      `gorm:"not null"`
    Author       string    `gorm:"not null"`
    Message      string    `gorm:"not null"`
    Date         time.Time `gorm:"not null"`
    FilesChanged int
    Additions    int
    Deletions    int
    Type         string    // feature, bugfix, refactor, etc.
    CreatedAt    time.Time
}
```

#### 3. WorkPoint (工作要点)
```go
type WorkPoint struct {
    ID          uint      `gorm:"primaryKey"`
    Title       string    `gorm:"not null"`
    Description string
    Category    string    `gorm:"not null"` // completed, ongoing, planned
    Priority    int       `gorm:"default:0"`
    Tags        string    // JSON array of tags
    Date        time.Time `gorm:"not null"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### 4. DailyReport (日报)
```go
type DailyReport struct {
    ID          uint      `gorm:"primaryKey"`
    Date        time.Time `gorm:"uniqueIndex;not null"`
    Title       string    `gorm:"not null"`
    Content     string    `gorm:"type:text"`
    Template    string    `gorm:"not null"`
    Status      string    `gorm:"default:draft"` // draft, published
    SentAt      *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 🔧 最新API文档

### 1. AI日报生成API
- `POST /api/v1/integrations/feishu/send/ai-gitlab-report` - AI生成GitLab日报并推送飞书
  ```json
  {
    "date": "2025-06-23",
    "work_items": ["工作成果1", "工作成果2"],
    "template": "standard",
    "message_type": "text"
  }
  ```

### 2. 飞书推送API
- `POST /api/v1/integrations/feishu/send/report` - 标准日报推送（混合数据源）
- `POST /api/v1/integrations/feishu/send/gitlab-report` - 纯GitLab数据日报推送
- `GET /api/v1/integrations/feishu/status` - 飞书集成状态

### 3. GitLab集成API
- `GET /api/v1/integrations/gitlab/status` - GitLab连接状态
- `POST /api/v1/integrations/gitlab/test` - 测试GitLab连接
- `GET /api/v1/integrations/gitlab/user` - 获取GitLab用户信息
- `GET /api/v1/integrations/gitlab/projects` - 获取GitLab项目列表
- `GET /api/v1/integrations/gitlab/commits/:date` - 获取指定日期提交记录

### 4. 工作要点管理API
- `GET /api/v1/points` - 获取工作要点列表
- `POST /api/v1/points` - 创建工作要点（带AI分析）
- `POST /api/v1/points/analyze` - 分析工作要点（不保存）
- `GET /api/v1/points/analysis` - 获取工作要点和分析信息

### 5. 日报管理API
- `POST /api/v1/reports/generate` - 生成标准日报
- `POST /api/v1/reports/gitlab/generate` - 生成GitLab专用日报
- `GET /api/v1/reports/preview` - 预览日报
- `GET /api/v1/reports/templates` - 获取可用模板

## 📝 配置示例

### config.yaml
```yaml
# 应用配置
app:
  name: "Daily Brief Generator"
  version: "1.0.0"
  port: 8080
  mode: "release" # debug, release

# 数据库配置
database:
  type: "sqlite"
  path: "./data/daily-brief.db"
  
# Git配置
git:
  scan_interval: "1h"
  max_commits_per_day: 50
  
# GitLab配置
gitlab:
  enabled: true
  base_url: "http://git.infoloop.cn"
  api_version: "v3"
  private_token: "your_gitlab_token"
  username: "your_username"
  user_email: "your_email@company.com"

# 飞书配置
feishu:
  enabled: true
  webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/your_webhook_token"
  app_id: "your_app_id"
  app_secret: "your_app_secret"
  chat_id: "your_chat_id"
  sign_secret: "your_sign_secret"

# AI配置
ai:
  enabled: true
  provider: "ollama"  # ollama or openai
  timeout: 30
  temperature: 0.7
  max_tokens: 2000
  # Ollama配置
  ollama_url: "http://localhost:11434"
  ollama_model: "deepseek-r1:7b"
  # OpenAI配置（可选）
  openai_url: "https://api.openai.com/v1/chat/completions"
  openai_model: "gpt-3.5-turbo"
  openai_key: "your_openai_key"

# 日报配置
report:
  default_template: "standard"
  timezone: "Asia/Shanghai"
  working_hours:
    start: "09:00"
    end: "18:00"
  ai:
    enabled: true
    style: "enterprise-api"
    
# 日志配置
logging:
  level: "info"
  format: "json"
  file: "./logs/app.log"
  max_size: 100 # MB
  max_age: 30   # days
```

## 🎨 日报模板示例

### 标准模板
```markdown
# 工作日报 - {{.Date}}

## 📈 今日完成
{{range .CompletedPoints}}
- {{.Title}}
  {{if .Description}}{{.Description}}{{end}}
{{end}}

## 💻 代码提交
{{range .Commits}}
- **{{.Repository}}**: {{.Message}} ({{.Author}})
  - 文件变更: {{.FilesChanged}} | +{{.Additions}} -{{.Deletions}}
{{end}}

## 🔄 进行中工作
{{range .OngoingPoints}}
- {{.Title}}
  {{if .Description}}{{.Description}}{{end}}
{{end}}

## 📋 明日计划
{{range .PlannedPoints}}
- {{.Title}}
  {{if .Description}}{{.Description}}{{end}}
{{end}}

---
*报告生成时间: {{.GeneratedAt}}*
```

## 🚦 开发规范

### 代码规范
1. **命名规范**：使用驼峰命名法，包名小写
2. **注释规范**：所有公开函数必须有注释，复杂逻辑添加详细注释
3. **错误处理**：统一错误处理机制，避免panic
4. **测试覆盖**：核心功能测试覆盖率达到80%以上

### Git提交规范
```
<type>(<scope>): <subject>

<body>

<footer>
```
- **type**: feat, fix, docs, style, refactor, test, chore
- **scope**: 影响范围
- **subject**: 简短描述

### 文档规范
1. 所有包都要有包文档
2. 复杂算法要有详细说明
3. API要有使用示例
4. 配置项要有说明

## 🎯 功能现状 & 扩展计划

### ✅ 已实现功能
1. **AI日报生成**：✅ 集成Ollama和OpenAI，自动生成自然语言日报
2. **GitLab深度集成**：✅ 支持GitLab API v3，智能提交记录获取
3. **飞书自动推送**：✅ 支持Webhook和API双模式推送
4. **多数据源融合**：✅ 用户要点与代码提交智能合并
5. **专业语言控制**：✅ AI提示词优化，避免AI化表达

### 🚧 进行中功能
1. **数据可视化**：工作统计图表和效率分析
2. **调度系统**：定时自动生成和推送
3. **移动端支持**：PWA应用和移动端适配

### 📋 近期扩展计划
1. **更多集成平台**：钉钉、企业微信、Slack
2. **更多AI模型**：支持通义千问、文心一言等
3. **项目管理集成**：Jira、Trello集成
4. **团队协作**：多人日报聚合和分析

### 🔮 长期规划
1. **智能分析**：基于AI的工作效率分析和建议
2. **云服务**：SaaS化部署和多租户支持
3. **企业版**：权限管理、审批流程、合规监控
4. **插件生态**：开放插件API，支持第三方扩展

## ✅ 成功指标

1. **功能完整性**：所有核心功能正常运行
2. **代码质量**：通过静态分析检查，测试覆盖率≥80%
3. **性能指标**：日报生成时间<5秒，支持≥10个仓库
4. **学习目标**：通过项目掌握Go语言核心概念和最佳实践

---

这个方案提供了一个完整的技术路线图，既能满足您的实际需求，又能作为学习Go语言的优秀项目。每个模块都有详细的设计和实现计划，代码将包含完备的注释来帮助学习。

准备好开始实施了吗？我们可以从第一阶段的基础框架搭建开始！ 