# 配置文件说明

## 配置文件结构

- `config.example.yaml` - 配置文件模板（可提交到版本控制）
- `config.yaml` - 实际配置文件（包含敏感信息，已被.gitignore忽略）
- `config.prod.yaml` - 生产环境配置（已被.gitignore忽略）
- `config.local.yaml` - 本地开发配置（已被.gitignore忽略）

## 初始化配置

1. 复制示例配置文件：
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   ```

2. 编辑 `configs/config.yaml` 并填入真实的配置信息：
   - GitLab服务器地址和认证信息
   - 飞书Webhook URL和密钥
   - AI服务配置（如需要）

## 重要提醒

⚠️ **绝对不要将包含真实密钥、密码、Token等敏感信息的配置文件提交到版本控制系统！**

所有包含敏感信息的配置文件都已被 `.gitignore` 忽略：
- `configs/config.yaml`
- `configs/config.prod.yaml`
- `configs/config.local.yaml`
- `configs/config.dev.yaml`

## 配置项说明

### GitLab配置
- `base_url`: GitLab服务器地址
- `username`: GitLab用户名
- `token`: Personal Access Token（推荐使用）
- `project_ids`: 要监控的项目ID列表

### 飞书配置
- `webhook_url`: 飞书机器人Webhook地址
- `app_secret`: 飞书应用密钥（Webhook模式不需要）

### AI配置
- `provider`: AI提供商（ollama、openai等）
- `ollama_url`: Ollama服务地址
- `ollama_model`: 使用的模型名称

更多详细配置说明请参考 `config.example.yaml` 文件中的注释。 