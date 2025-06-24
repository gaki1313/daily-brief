#!/bin/bash

# AI日报生成测试脚本

echo "🤖 Testing AI Report Generation..."

# 检查应用是否运行
if ! curl -s http://localhost:3000/health > /dev/null; then
    echo "❌ Application is not running. Please start with './main'"
    exit 1
fi

echo "✅ Application is running"

# 检查Ollama是否运行 (如果配置了Ollama)
if curl -s http://localhost:11434/api/version > /dev/null; then
    echo "✅ Ollama service is running"
    
    # 检查模型是否可用
    if ollama list | grep -q "deepseek-r1\|qwen2\|llama"; then
        AVAILABLE_MODEL=$(ollama list | grep -E "(deepseek-r1|qwen2|llama)" | head -1 | awk '{print $1}')
        echo "✅ AI model is available: $AVAILABLE_MODEL"
    else
        echo "⚠️  No AI model found. Please run: ollama pull deepseek-r1:7b"
    fi
else
    echo "⚠️  Ollama service not running. AI will fallback to template generation."
fi

# 获取今天的日期
TODAY=$(date +%Y-%m-%d)

echo "📅 Testing with date: $TODAY"

# 测试普通GitLab报告
echo ""
echo "🧪 Testing standard GitLab report..."
RESPONSE=$(curl -s -X POST http://localhost:3000/api/v1/integrations/feishu/send/gitlab-report \
     -H "Content-Type: application/json" \
     -d "{\"date\": \"$TODAY\"}")

if echo "$RESPONSE" | grep -q '"code":200'; then
    echo "✅ Standard GitLab report test passed"
else
    echo "❌ Standard GitLab report test failed"
    echo "Response: $RESPONSE"
fi

# 测试AI GitLab报告
echo ""
echo "🤖 Testing AI-powered GitLab report..."
AI_RESPONSE=$(curl -s -X POST http://localhost:3000/api/v1/integrations/feishu/send/ai-gitlab-report \
     -H "Content-Type: application/json" \
     -d "{\"date\": \"$TODAY\"}")

if echo "$AI_RESPONSE" | grep -q '"code":200'; then
    echo "✅ AI GitLab report test passed"
    
    # 检查是否真的使用了AI
    if echo "$AI_RESPONSE" | grep -q '"ai_enabled":true'; then
        echo "🎉 AI generation is working!"
    else
        echo "⚠️  AI is disabled or not available, using template fallback"
    fi
    
    # 显示响应信息
    echo "📊 Response details:"
    echo "$AI_RESPONSE" | jq '.' 2>/dev/null || echo "$AI_RESPONSE"
else
    echo "❌ AI GitLab report test failed"
    echo "Response: $AI_RESPONSE"
fi

echo ""
echo "🎯 Test Summary:"
echo "- Standard GitLab Report: $(echo "$RESPONSE" | grep -q '"code":200' && echo "✅ PASS" || echo "❌ FAIL")"
echo "- AI GitLab Report: $(echo "$AI_RESPONSE" | grep -q '"code":200' && echo "✅ PASS" || echo "❌ FAIL")"
echo "- AI Enabled: $(echo "$AI_RESPONSE" | grep -q '"ai_enabled":true' && echo "✅ YES" || echo "❌ NO")"

echo ""
echo "💡 Tips:"
echo "- If AI is not working, check Ollama service: ollama serve"
echo "- Download a model if needed: ollama pull deepseek-r1:7b"
echo "- Check logs for detailed error information"
echo "- Verify AI configuration in configs/config.local.yaml" 