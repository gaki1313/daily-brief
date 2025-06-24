#!/bin/bash

# GitLab Token Generator
# 用于通过用户名/密码生成Personal Access Token

set -e

GITLAB_URL="${1:-http://git.infoloop.cn}"
USERNAME="${2:-xia.jiaqi@qingniu.co}"
PASSWORD="${3:-Xjq19930208}"

echo "Generating GitLab Personal Access Token..."
echo "GitLab URL: $GITLAB_URL"
echo "Username: $USERNAME"
echo ""

# 获取CSRF token
echo "Step 1: Getting CSRF token..."
CSRF_TOKEN=$(curl -s -c cookies.txt "$GITLAB_URL/users/sign_in" | grep -o 'name="authenticity_token" value="[^"]*"' | cut -d'"' -f4)

if [ -z "$CSRF_TOKEN" ]; then
    echo "Failed to get CSRF token"
    exit 1
fi

echo "CSRF token: $CSRF_TOKEN"

# 登录
echo "Step 2: Logging in..."
LOGIN_RESPONSE=$(curl -s -b cookies.txt -c cookies.txt \
    -d "user[login]=$USERNAME" \
    -d "user[password]=$PASSWORD" \
    -d "authenticity_token=$CSRF_TOKEN" \
    -X POST \
    "$GITLAB_URL/users/sign_in")

# 检查登录是否成功
if echo "$LOGIN_RESPONSE" | grep -q "Invalid Login or password"; then
    echo "Login failed - Invalid credentials"
    rm -f cookies.txt
    exit 1
fi

echo "Login successful"

# 获取个人访问令牌页面的CSRF token
echo "Step 3: Getting token page CSRF..."
TOKEN_PAGE=$(curl -s -b cookies.txt "$GITLAB_URL/-/profile/personal_access_tokens")
TOKEN_CSRF=$(echo "$TOKEN_PAGE" | grep -o 'name="authenticity_token" value="[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN_CSRF" ]; then
    echo "Failed to get token page CSRF token"
    rm -f cookies.txt
    exit 1
fi

echo "Token CSRF: $TOKEN_CSRF"

# 创建Personal Access Token
echo "Step 4: Creating Personal Access Token..."
TOKEN_NAME="daily-brief-$(date +%Y%m%d-%H%M%S)"

TOKEN_RESPONSE=$(curl -s -b cookies.txt \
    -d "personal_access_token[name]=$TOKEN_NAME" \
    -d "personal_access_token[expires_at]=" \
    -d "personal_access_token[scopes][]=api" \
    -d "personal_access_token[scopes][]=read_api" \
    -d "personal_access_token[scopes][]=read_repository" \
    -d "authenticity_token=$TOKEN_CSRF" \
    -X POST \
    "$GITLAB_URL/-/profile/personal_access_tokens")

# 提取token
TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o 'id="created-personal-access-token"[^>]*value="[^"]*"' | cut -d'"' -f6)

if [ -z "$TOKEN" ]; then
    echo "Failed to create token. Response:"
    echo "$TOKEN_RESPONSE" | head -20
    rm -f cookies.txt
    exit 1
fi

echo ""
echo "✅ Personal Access Token created successfully!"
echo "Token: $TOKEN"
echo "Name: $TOKEN_NAME"
echo ""
echo "Now update your config file:"
echo "git:"
echo "  gitlab:"
echo "    enabled: true"
echo "    base_url: \"$GITLAB_URL\""
echo "    username: \"\""
echo "    password: \"\""
echo "    token: \"$TOKEN\""
echo ""

# 清理
rm -f cookies.txt

# 测试token
echo "Testing token..."
USER_INFO=$(curl -k -s -H "Authorization: Bearer $TOKEN" "$GITLAB_URL/api/v4/user")

if echo "$USER_INFO" | grep -q '"id"'; then
    echo "✅ Token works! User info:"
    echo "$USER_INFO" | grep -o '"username":"[^"]*"' | cut -d'"' -f4
else
    echo "❌ Token test failed"
    echo "Response: $USER_INFO"
fi 