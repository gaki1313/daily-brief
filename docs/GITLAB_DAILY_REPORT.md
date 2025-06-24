# GitLab 日报生成功能文档

## 概述

GitLab日报生成功能允许您从GitLab服务器自动收集提交记录，并生成专业的工作日报。该功能支持多种模板格式，可以与飞书等平台集成。

## 功能特性

- 🔍 **自动扫描**: 从GitLab API自动获取指定日期的提交记录
- 📊 **智能分析**: 分析提交类型、代码变更统计、工作时长等
- 📋 **多种模板**: 支持标准、简洁、详细、Markdown等多种报告模板
- 🔄 **实时数据**: 实时从GitLab服务器获取最新提交信息
- 💾 **数据同步**: 可选择将GitLab数据同步到本地数据库
- 🚀 **API接口**: 提供完整的REST API接口

## 配置要求

### 1. GitLab配置 (config.local.yaml)

```yaml
git:
  gitlab:
    enabled: true                           # 启用GitLab集成
    base_url: "http://git.infoloop.cn"      # GitLab服务器地址
    username: "your.email@company.com"      # GitLab用户名（可选）
    password: ""                            # GitLab密码（不推荐）
    token: "your_personal_access_token"     # Personal Access Token（推荐）
    project_ids: []                         # 指定项目ID（空表示所有项目）
```

### 2. Personal Access Token

GitLab Personal Access Token需要以下权限：
- `read_api` - 读取API数据
- `read_repository` - 访问仓库信息
- `read_user` - 读取用户信息

创建方法：
1. 登录GitLab
2. 进入 `Settings` → `Access Tokens`
3. 创建新token，选择上述权限
4. 将token配置到 `config.local.yaml`

## API接口

### 1. 预览GitLab日报

```bash
GET /api/v1/reports/gitlab/preview?date=2024-01-15&template=standard
```

**参数:**
- `date`: 日期 (YYYY-MM-DD格式，可选，默认今天)
- `template`: 模板名称 (可选，默认"standard")

**返回:**
```json
{
    "code": 200,
    "message": "GitLab report preview generated successfully",
    "data": {
        "content": "报告内容...",
        "date": "2024-01-15",
        "template": "standard"
    }
}
```

### 2. 生成并保存GitLab日报

```bash
POST /api/v1/reports/gitlab/generate
Content-Type: application/json

{
    "date": "2024-01-15",
    "template": "standard"
}
```

**返回:**
```json
{
    "code": 200,
    "message": "GitLab daily report generated successfully",
    "data": {
        "id": 123,
        "date": "2024-01-15T00:00:00Z",
        "title": "GitLab Daily Brief - 2024年01月15日",
        "content": "报告内容...",
        "template": "standard",
        "status": "draft"
    }
}
```

### 3. 获取GitLab日报数据

```bash
GET /api/v1/reports/gitlab/data/2024-01-15
```

**返回:**
```json
{
    "code": 200,
    "message": "GitLab report data retrieved successfully",
    "data": {
        "date": "2024-01-15T00:00:00Z",
        "title": "GitLab Daily Brief - 2024年01月15日",
        "summary": {
            "total_commits": 5,
            "lines_added": 120,
            "lines_deleted": 45,
            "files_changed": 8,
            "productivity_score": 85.5
        },
        "commits": [...],
        "statistics": {...},
        "metadata": {...}
    }
}
```

## 使用示例

### 1. 命令行测试

使用提供的测试脚本：

```bash
# 基本测试
./scripts/test_gitlab_report.sh

# 指定参数
./scripts/test_gitlab_report.sh http://localhost:3000 2024-01-15 markdown
```

### 2. cURL示例

```bash
# 预览今天的日报
curl "http://localhost:3000/api/v1/reports/gitlab/preview"

# 生成指定日期的日报
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"date":"2024-01-15","template":"standard"}' \
  "http://localhost:3000/api/v1/reports/gitlab/generate"

# 获取日报数据
curl "http://localhost:3000/api/v1/reports/gitlab/data/2024-01-15"
```

### 3. JavaScript示例

```javascript
// 预览GitLab日报
async function previewGitLabReport(date = null, template = 'standard') {
    const url = new URL('/api/v1/reports/gitlab/preview', 'http://localhost:3000');
    if (date) url.searchParams.set('date', date);
    url.searchParams.set('template', template);
    
    const response = await fetch(url);
    const result = await response.json();
    return result.data.content;
}

// 生成GitLab日报
async function generateGitLabReport(date, template = 'standard') {
    const response = await fetch('/api/v1/reports/gitlab/generate', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ date, template })
    });
    
    const result = await response.json();
    return result.data;
}

// 使用示例
previewGitLabReport('2024-01-15', 'markdown')
    .then(content => console.log(content));
```

## 报告模板

### 1. Standard模板

包含完整的统计信息、提交记录、工作要点等，适合日常使用。

### 2. Simple模板

简洁版本，只显示核心数据和重点信息。

### 3. Detailed模板

详细版本，包含所有可用信息和完整的元数据。

### 4. Markdown模板

Markdown格式，适合技术团队和文档平台。

## 故障排除

### 1. GitLab连接失败

**问题**: `GitLab integration is not enabled or configured`

**解决方法**:
- 检查 `config.local.yaml` 中的GitLab配置
- 确保 `enabled: true`
- 验证 `base_url` 和 `token` 配置正确

### 2. 认证失败

**问题**: `401 Unauthorized`

**解决方法**:
- 检查Personal Access Token是否有效
- 确认token具有必要的权限
- 验证GitLab服务器地址是否正确

### 3. 无提交记录

**问题**: 日报显示0个提交

**解决方法**:
- 确认指定日期确实有提交记录
- 检查GitLab用户邮箱是否与提交者邮箱匹配
- 验证项目权限设置

### 4. API响应慢

**解决方法**:
- 如果项目较多，考虑在配置中指定 `project_ids`
- 检查GitLab服务器网络连接
- 考虑启用本地数据同步功能

## 最佳实践

1. **Token安全**: 使用Personal Access Token而不是密码
2. **权限最小化**: 只授予必要的API权限
3. **项目过滤**: 大型组织建议指定关注的项目ID
4. **定期同步**: 可以设置定时任务批量同步GitLab数据
5. **模板选择**: 根据不同场景选择合适的报告模板

## 与飞书集成

GitLab日报可以直接发送到飞书群聊：

```bash
# 生成并发送到飞书
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"date":"2024-01-15","template":"simple"}' \
  "http://localhost:3000/api/v1/integrations/feishu/send/report"
```

详见 [飞书集成文档](./FEISHU_INTEGRATION.md)。 