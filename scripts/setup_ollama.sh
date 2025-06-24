#!/bin/bash

# Ollama安装和配置脚本
# 用于设置AI日报生成功能

echo "🤖 Setting up Ollama for AI-powered daily reports..."

# 检查是否已安装Ollama
if command -v ollama &> /dev/null; then
    echo "✅ Ollama already installed"
else
    echo "📦 Installing Ollama..."
    
    # 检测操作系统
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        echo "🍎 Detected macOS, installing via curl..."
        curl -fsSL https://ollama.ai/install.sh | sh
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        echo "🐧 Detected Linux, installing via curl..."
        curl -fsSL https://ollama.ai/install.sh | sh
    else
        echo "❌ Unsupported OS. Please install Ollama manually from https://ollama.ai"
        exit 1
    fi
fi

# 启动Ollama服务
echo "🚀 Starting Ollama service..."
ollama serve &
OLLAMA_PID=$!

# 等待服务启动
echo "⏳ Waiting for Ollama service to start..."
sleep 5

# 检查服务是否运行
if ! curl -s http://localhost:11434/api/version > /dev/null; then
    echo "❌ Failed to start Ollama service"
    exit 1
fi

echo "✅ Ollama service started successfully"

# 检查是否已有可用模型
AVAILABLE_MODELS=$(ollama list 2>/dev/null | grep -E "(deepseek-r1|qwen2|llama)" | head -1)

if [ -n "$AVAILABLE_MODELS" ]; then
    echo "✅ Found existing AI model: $AVAILABLE_MODELS"
    MODEL_NAME=$(echo "$AVAILABLE_MODELS" | awk '{print $1}')
else
    # 下载推荐的模型
    echo "📥 Downloading recommended AI model: deepseek-r1:7b..."
    echo "⚠️  This may take several minutes depending on your internet speed..."

    ollama pull deepseek-r1:7b

    if [ $? -eq 0 ]; then
        echo "✅ Model deepseek-r1:7b downloaded successfully"
        MODEL_NAME="deepseek-r1:7b"
    else
        echo "❌ Failed to download deepseek-r1:7b. Trying alternative model..."
        ollama pull qwen2:7b
        if [ $? -eq 0 ]; then
            echo "✅ Alternative model qwen2:7b downloaded successfully"
            MODEL_NAME="qwen2:7b"
        else
            echo "❌ Failed to download qwen2:7b. Trying llama3.1:8b..."
            ollama pull llama3.1:8b
            if [ $? -eq 0 ]; then
                echo "✅ Alternative model llama3.1:8b downloaded successfully"
                MODEL_NAME="llama3.1:8b"
            else
                echo "❌ Failed to download any model"
                exit 1
            fi
        fi
    fi
fi

# 测试模型
echo "🧪 Testing AI model: $MODEL_NAME..."
TEST_RESPONSE=$(ollama run "$MODEL_NAME" "请用中文简单介绍一下你自己" --timeout 30s 2>/dev/null)

if [ -n "$TEST_RESPONSE" ]; then
    echo "✅ AI model test successful!"
    echo "📝 Test response: $TEST_RESPONSE"
else
    echo "⚠️  AI model test failed, but installation completed"
fi

echo ""
echo "🎉 Ollama setup completed!"
echo ""
echo "📋 Next steps:"
echo "1. Ensure Ollama is running: ollama serve"
echo "2. Your daily-brief config should have:"
echo "   ai:"
echo "     enabled: true"
echo "     provider: ollama"
echo "     ollama_url: http://localhost:11434"
echo "     ollama_model: $MODEL_NAME"
echo ""
echo "3. Test the AI report generation with:"
echo "   curl -X POST http://localhost:3000/api/v1/integrations/feishu/send/ai-gitlab-report \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"date\": \"$(date +%Y-%m-%d)\"}'"
echo ""
echo "🔧 Available models:"
ollama list

echo ""
echo "💡 Tips:"
echo "- Use 'ollama run MODEL_NAME' to test models interactively"
echo "- Use 'ollama ps' to see running models"
echo "- Use 'ollama stop MODEL_NAME' to stop a model"
echo "- Ollama will auto-start on system boot" 