# 🚀 飞书集成指南

这个文档将指导您如何配置和使用Daily Brief的飞书集成功能，实现自动发送日报到飞书群聊。

## 📋 功能特性

- ✅ **多种发送方式**：支持API发送和Webhook发送
- ✅ **智能消息格式化**：自动将日报转换为飞书支持的消息格式
- ✅ **自动定时发送**：支持配置自动发送时间
- ✅ **手动发送控制**：提供API手动触发发送
- ✅ **发送状态跟踪**：记录发送历史和状态
- ✅ **错误重试机制**：自动处理网络异常和重试
- ✅ **多种消息类型**：支持文本消息和卡片消息

## 🔧 配置方法

### 方式一：应用API模式（推荐）

1. **创建飞书应用**
   - 访问 [飞书开发者后台](https://open.feishu.cn/app)
   - 创建企业自建应用
   - 获取 App ID 和 App Secret

2. **配置权限**
   - 在应用管理页面，添加以下权限：
     - `im:message`：发送消息
     - `im:message:send_as_bot`：以机器人身份发送

3. **获取群聊ID**
   - 将机器人添加到目标群聊
   - 使用飞书API获取群聊ID，或通过网页版飞书URL获取

4. **配置应用**
   ```yaml
   feishu:
     enabled: true
     app_id: "cli_xxx"           # 你的应用ID
     app_secret: "xxx"           # 你的应用密钥
     chat_id: "oc_xxx"          # 群聊ID
   ```

### 方式二：Webhook模式（简单）

1. **创建群机器人**
   - 在飞书群聊中，点击 "设置" → "群机器人"
   - 添加自定义机器人
   - 复制Webhook地址

2. **配置应用**
   ```yaml
   feishu:
     enabled: true
     webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
   ```

### 环境变量配置

为了安全起见，建议使用环境变量设置敏感信息：

```bash
# 飞书应用配置
export DAILY_BRIEF_FEISHU_ENABLED=true
export DAILY_BRIEF_FEISHU_APP_ID="cli_xxx"
export DAILY_BRIEF_FEISHU_APP_SECRET="xxx"
export DAILY_BRIEF_FEISHU_CHAT_ID="oc_xxx"

# 或者使用Webhook
export DAILY_BRIEF_FEISHU_WEBHOOK_URL="https://open.feishu.cn/open-apis/bot/v2/hook/xxx"
```

## ⚙️ 自动发送配置

配置自动定时发送日报：

```yaml
report:
  auto_send: true          # 启用自动发送
  send_time: "18:00"       # 发送时间（24小时制）
  timezone: "Asia/Shanghai" # 时区设置
```

## 🎯 API使用指南

### 1. 查看集成状态

```bash
curl http://localhost:3000/api/v1/integrations/feishu/status
```

响应示例：
```json
{
  "code": 200,
  "message": "Feishu status retrieved successfully",
  "data": {
    "enabled": true,
    "is_running": true,
    "stats": {
      "enabled": true,
      "has_app_id": true,
      "has_app_secret": true,
      "has_chat_id": true,
      "token_valid": true
    },
    "scheduler": {
      "total_send_count": 5,
      "success_count": 5,
      "failure_count": 0,
      "next_send_time": "2025-06-24T18:00:00+08:00"
    }
  }
}
```

### 2. 测试连接

```bash
curl -X POST http://localhost:3000/api/v1/integrations/feishu/test
```

### 3. 手动发送日报

```bash
curl -X POST http://localhost:3000/api/v1/integrations/feishu/send/report \
  -H "Content-Type: application/json" \
  -d '{
    "date": "2025-06-23",
    "template": "standard",
    "message_type": "text"
  }'
```

### 4. 发送自定义消息

```bash
curl -X POST http://localhost:3000/api/v1/integrations/feishu/send \
  -H "Content-Type: application/json" \
  -d '{
    "message_type": "text",
    "content": "这是一条测试消息"
  }'
```

### 5. 调度器管理

启动自动调度器：
```bash
curl -X POST http://localhost:3000/api/v1/integrations/feishu/scheduler/start
```

停止自动调度器：
```bash
curl -X POST http://localhost:3000/api/v1/integrations/feishu/scheduler/stop
```

测试调度器发送：
```bash
curl -X POST http://localhost:3000/api/v1/integrations/feishu/scheduler/test
```

### 6. 查看发送日志

```bash
curl http://localhost:3000/api/v1/integrations/logs?platform=feishu&limit=10
```

## 📱 消息格式示例

### 文本消息格式
```
📊 Daily Brief - 2025年06月23日
==============================

📈 今日概览
• 完成任务：5/7 (71.4%)
• 代码提交：3 次
• 代码变更：+156 -42 行
• 生产力评分：75.2/100

⭐ 重点任务
• 完成飞书集成开发
• 修复日报生成Bug
• 更新项目文档

💻 代码提交
✨ feat: 添加飞书集成功能 (14:30)
🐛 fix: 修复配置加载问题 (15:45)
📚 docs: 更新集成文档 (16:20)

📝 16:30 自动生成
```

### 卡片消息格式
飞书卡片消息会显示为结构化的卡片，包含：
- 标题和日期
- 统计数据（网格布局）
- 重点任务列表
- 代码提交历史
- 生成时间

## 🔍 故障排除

### 常见问题

1. **发送失败："Feishu integration is not enabled"**
   - 检查配置文件中 `feishu.enabled` 是否为 `true`
   - 确认必要的认证信息已正确设置

2. **发送失败："Chat ID not configured"**
   - 确认已设置正确的群聊ID
   - 检查机器人是否已添加到目标群聊

3. **发送失败："failed to get access token"**
   - 检查App ID和App Secret是否正确
   - 确认应用权限配置正确

4. **Webhook发送失败**
   - 检查Webhook URL是否正确
   - 确认机器人在群聊中处于活跃状态

### 调试技巧

1. **查看详细日志**
   ```bash
   tail -f logs/app.log | grep feishu
   ```

2. **检查集成状态**
   ```bash
   curl http://localhost:3000/api/v1/integrations/feishu/status
   ```

3. **测试连接**
   ```bash
   curl -X POST http://localhost:3000/api/v1/integrations/feishu/test
   ```

## 🔐 安全建议

1. **使用环境变量**：不要在配置文件中明文保存App Secret
2. **定期轮换密钥**：定期更新应用密钥
3. **最小权限原则**：只授予必要的API权限
4. **监控使用**：定期检查发送日志和状态

## 📈 最佳实践

1. **测试优先**：在生产环境部署前，先在测试群聊中验证
2. **渐进部署**：先手动发送测试，再启用自动发送
3. **监控报警**：设置发送失败的监控报警
4. **备用方案**：准备多种通知方式作为备用

## 🔄 升级和维护

当需要升级飞书集成功能时：

1. 停止服务器
2. 更新代码
3. 重新构建应用
4. 验证配置文件
5. 启动服务器
6. 测试功能

```bash
# 升级步骤
pkill -f "daily-brief"
git pull
go build -o main cmd/main.go
./main &

# 验证
curl http://localhost:3000/api/v1/integrations/feishu/status
```

## 飞书集成说明

## 概述
本项目支持与飞书机器人集成，可以自动发送日报到飞书群聊。支持两种集成模式：
1. **Webhook模式**：通过群机器人的Webhook地址发送消息（推荐）
2. **API模式**：通过飞书开放平台API发送消息

## Webhook模式配置

### 1. 创建飞书机器人
1. 在飞书群聊中，点击群设置 → 群机器人 → 添加机器人
2. 选择"自定义机器人"
3. 配置机器人基本信息（名称、头像、描述）
4. **重要：安全设置**
   - 如果选择"签名校验"，请记录下签名密钥
   - 也可选择"关键词"或"IP白名单"等其他验证方式
   - 或选择"暂不设置"（不推荐用于生产环境）

### 2. 获取Webhook信息
完成机器人创建后，你会获得：
- **Webhook URL**：形如 `https://open.feishu.cn/open-apis/bot/v2/hook/xxx`
- **签名密钥**：如果启用了签名校验（如果没有启用则为空）

### 3. 配置项目
在 `configs/config.local.yaml` 中配置：

```yaml
feishu:
  enabled: true
  webhook_url: "你的webhook地址"
  webhook_secret: "你的签名密钥"  # 如果启用了签名校验
```

## 问题排查

### 错误：sign match fail or timestamp is not within one hour from current time

**原因分析：**
1. **签名密钥缺失或错误**：机器人配置了签名校验，但项目中未配置或配置错误
2. **时间戳问题**：服务器时间与飞书服务器时间相差超过1小时

**解决方案：**

#### 方案1：配置正确的签名密钥
1. 在飞书机器人设置中查看签名密钥
2. 将密钥正确配置到 `webhook_secret` 字段

#### 方案2：修改机器人安全设置
1. 进入群聊设置 → 群机器人 → 找到你的机器人 → 点击设置
2. 修改安全设置：
   - 改为"关键词"验证（需要消息包含特定关键词）
   - 改为"IP白名单"（需要配置服务器IP）
   - 改为"暂不设置"（最简单但不安全）
3. 如果改为非签名验证，将配置中的 `webhook_secret` 设为空字符串

#### 方案3：检查时间同步
```bash
# 检查服务器时间
date
# 与网络时间同步（如果需要）
sudo ntpdate -s time.nist.gov
```

### 测试配置
使用以下命令测试配置：

```bash
# 测试飞书连接
curl -X GET "http://localhost:3000/api/v1/integrations/feishu/status"

# 发送测试消息
curl -X POST "http://localhost:3000/api/v1/integrations/feishu/send" \
  -H "Content-Type: application/json" \
  -d '{
    "msg_type": "text", 
    "content": {"text": "测试消息"}
  }'
```

## API模式配置

如果你偏好使用飞书开放平台API（适用于更复杂的场景）：

### 1. 创建飞书应用
1. 访问[飞书开放平台](https://open.feishu.cn/)
2. 创建企业自建应用
3. 获取 App ID 和 App Secret
4. 配置应用权限和事件订阅

### 2. 配置项目
```yaml
feishu:
  enabled: true
  app_id: "你的应用ID"
  app_secret: "你的应用密钥"
  chat_id: "群聊ID"
```

## 功能特性

### 支持的消息类型
- 文本消息
- 富文本消息
- 卡片消息（日报专用格式）

### 自动发送功能
配置自动发送日报：

```yaml
report:
  auto_send: true
  send_time: "18:30"  # 每天18:30自动发送
```

### 多模板支持
- `standard`：标准格式日报
- `simple`：简洁格式
- `detailed`：详细格式
- `markdown`：Markdown格式

## API接口

### 发送自定义消息
```http
POST /api/v1/integrations/feishu/send
Content-Type: application/json

{
  "msg_type": "text",
  "content": {
    "text": "你的消息内容"
  }
}
```

### 发送日报
```http
POST /api/v1/integrations/feishu/send/report
Content-Type: application/json

{
  "date": "2025-01-21",
  "template": "standard"
}
```

### 获取发送统计
```http
GET /api/v1/integrations/feishu/stats
```

## 注意事项

1. **安全性**：生产环境建议启用签名校验或IP白名单
2. **频率限制**：飞书对机器人消息有频率限制，避免短时间内大量发送
3. **权限**：确保机器人有发送消息到目标群聊的权限
4. **网络**：确保服务器能访问飞书API（`open.feishu.cn`）

## 最佳实践

1. **测试环境**：先在测试群中配置和测试机器人
2. **错误处理**：监控集成日志，及时处理发送失败的情况
3. **内容优化**：根据团队需求调整日报模板和内容格式
4. **备份方案**：配置多个通知渠道，避免单点故障

```bash
# 升级步骤
pkill -f "daily-brief"
git pull
go build -o main cmd/main.go
./main &

# 验证
curl http://localhost:3000/api/v1/integrations/feishu/status
``` 