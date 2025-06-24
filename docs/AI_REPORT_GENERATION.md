# AI智能日报生成功能

## 概述

Daily Brief Generator现在支持使用大语言模型(LLM)自动生成智能化的工作日报。AI会分析你的Git提交记录，自动生成结构化、专业的中文工作日报。

## 支持的AI提供商

### 1. Ollama (推荐) 🌟
- **完全免费**，本地运行
- **隐私安全**，代码信息不会上传到外部服务器
- **支持多种模型**：qwen2、llama3.1、gemma2等
- **中文支持好**

### 2. OpenAI
- 需要API密钥
- 每月有免费额度
- 支持代理配置

### 3. Google Gemini (计划中)
- 每天有免费调用次数
- 性能优秀

### 4. 阿里通义千问 (计划中)
- 中文支持优秀
- 每天有免费额度

## 快速开始

### 步骤1：安装Ollama (推荐)

```bash
# 运行自动安装脚本
./scripts/setup_ollama.sh
```

或手动安装：

```bash
# macOS
curl -fsSL https://ollama.ai/install.sh | sh

# 启动服务
ollama serve

# 下载中文模型
ollama pull deepseek-r1:7b
```

### 步骤2：配置AI

编辑 `configs/config.local.yaml`：

```yaml
report:
  ai:
    enabled: true
    provider: "ollama"
    ollama_url: "http://localhost:11434"
    ollama_model: "deepseek-r1:7b"
    temperature: 0.7
    max_tokens: 2000
    timeout: 60
```

### 步骤3：启动应用

```bash
./main
```

### 步骤4：生成AI日报

```bash
# 生成并发送AI日报到飞书
curl -X POST http://localhost:3000/api/v1/integrations/feishu/send/ai-gitlab-report \
     -H "Content-Type: application/json" \
     -d '{"date": "2025-06-23"}'
```

## API接口

### AI GitLab日报接口

**POST** `/api/v1/integrations/feishu/send/ai-gitlab-report`

生成AI智能GitLab日报并发送到飞书。

**请求参数：**
```json
{
  "date": "2025-06-23",          // 必需：日期 (YYYY-MM-DD)
  "template": "standard",        // 可选：模板名称
  "message_type": "text"         // 可选：消息类型 (text/card)
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "AI GitLab report sent to Feishu successfully",
  "data": {
    "message_id": "",
    "sent_at": "2025-06-23T10:41:22+08:00",
    "duration": 954,
    "data_source": "gitlab-ai",
    "ai_enabled": true
  }
}
```

## AI生成的日报格式

AI会根据你的Git提交记录生成结构化的日报，包含：

### 1. 工作概述
简要总结当天的主要工作内容

### 2. 具体工作项
- 根据提交信息自动归类整理
- 突出重要的功能开发、bug修复、性能优化
- 提取业务价值和技术亮点

### 3. 技术亮点
自动识别技术相关的重要改进

### 4. 工作量统计
- 提交次数
- 代码变更统计 (+行数, -行数)
- 涉及文件数量

## 配置选项详解

```yaml
report:
  ai:
    enabled: true                    # 是否启用AI生成
    provider: "ollama"               # AI提供商
    
    # Ollama配置
    ollama_url: "http://localhost:11434"
    ollama_model: "deepseek-r1:7b"   # 推荐中文模型
    
    # OpenAI配置 (可选)
    openai_key: "your-api-key"
    openai_model: "gpt-3.5-turbo"
    openai_url: ""                   # 支持代理
    
    # 生成参数
    temperature: 0.7                 # 创造性 (0.0-1.0)
    max_tokens: 2000                 # 最大输出长度
    timeout: 60                      # 超时时间(秒)
```

## 推荐模型

### Ollama模型推荐

1. **deepseek-r1:7b** (推荐)
   - 中文支持优秀
   - 大小适中 (~4GB)
   - 生成质量高

2. **llama3.1:8b**
   - 英文优秀，中文良好
   - 大小适中 (~4.7GB)
   - 通用性强

3. **gemma2:9b**
   - 性能均衡
   - 大小适中 (~5.4GB)

### 模型选择建议

- **内存 < 8GB**: 使用 `qwen2:1.5b` 或 `llama3.1:3b`
- **内存 8-16GB**: 使用 `deepseek-r1:7b` 或 `qwen2:7b` (推荐)
- **内存 > 16GB**: 使用 `qwen2:14b` 或 `llama3.1:70b`

## 故障排除

### 1. AI生成失败

**症状**: 接口返回成功但使用模板生成
**解决方案**:
```bash
# 检查Ollama服务
curl http://localhost:11434/api/version

# 检查模型是否下载
ollama list

# 测试模型
ollama run deepseek-r1:7b "你好"
```

### 2. 模型下载慢

**解决方案**:
```bash
# 使用国内镜像 (如果可用)
export OLLAMA_HOST=0.0.0.0:11434

# 或使用更小的模型
ollama pull qwen2:1.5b
```

### 3. 内存不足

**解决方案**:
```bash
# 使用更小的模型
ollama pull qwen2:1.5b

# 更新配置
# ollama_model: "qwen2:1.5b"
```

### 4. 生成内容质量不佳

**解决方案**:
- 调整 `temperature` 参数 (0.5-0.9)
- 尝试不同的模型
- 确保提交信息写得清晰

## 性能优化

### 1. 模型预加载

```bash
# 预加载模型到内存
ollama run deepseek-r1:7b ""
```

### 2. 配置优化

```yaml
ai:
  temperature: 0.7     # 平衡创造性和准确性
  max_tokens: 1500     # 适中的输出长度
  timeout: 30          # 合理的超时时间
```

## 安全和隐私

### Ollama (本地部署)
- ✅ **完全本地运行**，代码信息不会离开你的机器
- ✅ **无需API密钥**，完全免费
- ✅ **离线可用**，不依赖网络

### 云端API
- ⚠️ **数据会发送到第三方服务器**
- ⚠️ **需要API密钥管理**
- ⚠️ **可能产生费用**

## 最佳实践

1. **写好提交信息**: AI的生成质量很大程度上取决于你的Git提交信息质量
2. **定期使用**: 每天下班前生成一次日报，养成好习惯
3. **模型选择**: 根据你的硬件配置选择合适的模型
4. **备份方案**: 配置模板作为AI失败时的备选方案

## 示例输出

### 输入 (Git提交记录)
```
1. fix: handle duplicate company name in settlement entities map
2. Merge branch 'hotfix/p2-tax-import-error' into 'qa/2025-06-23'
```

### 输出 (AI生成的日报)
```markdown
# GitLab Daily Brief - 2025年06月23日

## 工作概述
今天主要专注于税务导入功能的bug修复工作，解决了结算实体映射中的重复公司名称处理问题，并完成了相关代码的合并。

## 具体工作项

### 🐛 Bug修复
- **结算实体映射优化**: 修复了在处理结算实体映射时遇到重复公司名称的问题
  - 改进了重复名称的处理逻辑
  - 确保数据映射的准确性和唯一性

### 🔄 代码集成
- **分支合并**: 完成了税务导入错误修复分支的合并工作
  - 将 `hotfix/p2-tax-import-error` 分支合并到 `qa/2025-06-23`
  - 确保修复代码顺利进入测试环境

## 技术亮点
- 优化了数据处理逻辑，提升了系统的稳定性
- 通过热修复分支快速响应生产环境问题

## 工作量统计
- 提交次数: 2次
- 代码变更: +15行, -8行  
- 涉及文件: 3个
```

## 更多信息

- [Ollama官网](https://ollama.ai)
- [支持的模型列表](https://ollama.ai/library)
- [配置最佳实践](CONFIG_BEST_PRACTICES.md) 