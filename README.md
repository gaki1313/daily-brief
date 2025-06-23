# 日报生成器 (Daily Brief Generator)

一个基于Go语言开发的智能日报生成器，能够自动收集Git提交记录和用户要点，生成专业的工作日报，并通过飞书机器人推送。

## 🎯 项目目标

- **自动化日报生成**：基于Git提交记录和用户输入要点自动生成日报
- **多平台集成**：支持飞书等第三方平台推送
- **模块化设计**：易于扩展和维护的架构
- **学习导向**：通过完备的代码注释帮助Go语言学习

## 🏗️ 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   数据收集层     │    │   处理生成层     │    │   推送集成层     │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ • Git仓库扫描   │    │ • 模板引擎      │    │ • 飞书机器人     │
│ • 要点输入      │    │ • 内容生成      │    │ • 邮件推送       │
│ • 配置管理      │    │ • 格式化处理    │    │ • Webhook       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   存储管理层     │
                    ├─────────────────┤
                    │ • SQLite数据库  │
                    │ • 配置文件      │
                    │ • 日志管理      │
                    └─────────────────┘
```

## 📋 功能模块

### 1. 核心功能模块

#### 1.1 Git数据收集器 (Git Collector)
- **功能**：扫描指定Git仓库的提交记录
- **特性**：
  - 支持多仓库扫描
  - 按日期范围筛选提交
  - 提取提交信息（作者、时间、消息、文件变更）
  - 智能分类提交类型（feature、bugfix、refactor等）

#### 1.2 要点管理器 (Point Manager)
- **功能**：管理用户手动输入的工作要点
- **特性**：
  - 分类管理（完成项、进行中、计划项）
  - 优先级设置
  - 标签系统
  - 历史记录追踪

#### 1.3 日报生成引擎 (Report Generator)
- **功能**：基于收集的数据生成专业日报
- **特性**：
  - 多种日报模板
  - 智能内容排序
  - Markdown/HTML输出
  - 自定义格式支持

#### 1.4 第三方集成层 (Integration Layer)
- **功能**：与外部平台集成
- **特性**：
  - 飞书机器人集成
  - Webhook支持
  - 邮件推送
  - 可扩展的插件架构

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
- **Git操作**：go-git (纯Go实现的Git库)
- **模板引擎**：text/template + html/template
- **HTTP客户端**：resty (HTTP请求库)

### 第三方集成
- **飞书开放平台API**
- **Git仓库API** (GitHub, GitLab, Gitee等)

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
│   ├── config/           # 配置管理
│   │   ├── config.go     # 配置结构定义
│   │   └── loader.go     # 配置加载器
│   ├── collector/        # 数据收集器
│   │   ├── git.go        # Git数据收集
│   │   └── points.go     # 要点收集
│   ├── generator/        # 日报生成器
│   │   ├── engine.go     # 生成引擎
│   │   ├── template.go   # 模板管理
│   │   └── formatter.go  # 格式化处理
│   ├── integration/      # 第三方集成
│   │   ├── feishu.go     # 飞书集成
│   │   └── webhook.go    # Webhook集成
│   ├── storage/          # 数据存储
│   │   ├── models.go     # 数据模型
│   │   ├── database.go   # 数据库操作
│   │   └── migration.go  # 数据库迁移
│   └── server/           # HTTP服务器
│       ├── handlers.go   # 路由处理器
│       ├── middleware.go # 中间件
│       └── routes.go     # 路由定义
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

## 🔧 核心API设计

### 1. 仓库管理
- `GET /api/repositories` - 获取仓库列表
- `POST /api/repositories` - 添加新仓库
- `PUT /api/repositories/:id` - 更新仓库信息
- `DELETE /api/repositories/:id` - 删除仓库

### 2. 要点管理
- `GET /api/points` - 获取要点列表
- `POST /api/points` - 创建新要点
- `PUT /api/points/:id` - 更新要点
- `DELETE /api/points/:id` - 删除要点

### 3. 日报管理
- `GET /api/reports` - 获取日报列表
- `POST /api/reports/generate` - 生成日报
- `GET /api/reports/:date` - 获取指定日期日报
- `POST /api/reports/:id/send` - 发送日报

### 4. 集成管理
- `POST /api/integrations/feishu/send` - 发送到飞书
- `GET /api/integrations/status` - 获取集成状态

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
  
# 飞书配置
feishu:
  app_id: "your_app_id"
  app_secret: "your_app_secret"
  webhook_url: "your_webhook_url"
  enabled: true

# 日报配置
report:
  default_template: "standard"
  timezone: "Asia/Shanghai"
  working_hours:
    start: "09:00"
    end: "18:00"
    
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

## 🎯 扩展计划

### 近期扩展
1. **更多集成平台**：钉钉、企业微信、Slack
2. **AI增强**：基于AI的日报内容优化
3. **数据可视化**：工作统计图表
4. **移动端支持**：PWA应用

### 长期规划
1. **团队协作**：多人日报聚合
2. **项目管理集成**：Jira、Trello集成
3. **智能分析**：工作效率分析
4. **云服务**：SaaS化部署

## ✅ 成功指标

1. **功能完整性**：所有核心功能正常运行
2. **代码质量**：通过静态分析检查，测试覆盖率≥80%
3. **性能指标**：日报生成时间<5秒，支持≥10个仓库
4. **学习目标**：通过项目掌握Go语言核心概念和最佳实践

---

这个方案提供了一个完整的技术路线图，既能满足您的实际需求，又能作为学习Go语言的优秀项目。每个模块都有详细的设计和实现计划，代码将包含完备的注释来帮助学习。

准备好开始实施了吗？我们可以从第一阶段的基础框架搭建开始！ 