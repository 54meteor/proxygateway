#!/bin/bash
# AI Gateway 联调测试 - Phase 2: 用户 + API Key 初始化
# 执行方式: bash phase2_init_user.sh

set -e

BASE_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "Phase 2: 用户 + API Key 初始化"
echo "=========================================="

# 2.1 创建测试用户
echo -e "\n${YELLOW}[2.1] 创建测试用户${NC}"
RESP=$(curl -s -X POST http://localhost:8080/debug/init)
echo "响应: $RESP"

if echo "$RESP" | grep -q '"user_id"'; then
    echo -e "${GREEN}✓ 用户创建成功${NC}"
    USER_ID=$(echo "$RESP" | grep -o '"user_id":"[^"]*"' | cut -d'"' -f4)
    API_KEY=$(echo "$RESP" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
    echo "User ID: $USER_ID"
    echo "API Key: $API_KEY"
    
    # 保存到文件供后续阶段使用
    cat > /tmp/test_api_key.txt << EOF
API_KEY=$API_KEY
USER_ID=$USER_ID
EOF
    echo -e "${GREEN}✓ API Key 已保存到 /tmp/test_api_key.txt${NC}"
else
    echo -e "${RED}✗ 用户创建失败${NC}"
    exit 1
fi

# 2.2 保存 Key 信息
echo -e "\n${YELLOW}[2.2] 验证 Key 格式${NC}"
if echo "$API_KEY" | grep -qE '^[0-9a-f-]{36}$'; then
    echo -e "${GREEN}✓ API Key 格式正确 (UUID)${NC}"
else
    echo -e "${RED}✗ API Key 格式异常${NC}"
fi

echo -e "\n=========================================="
echo "Phase 2 完成"
echo "=========================================="
echo "下一步: 使用 API_KEY 进行 Phase 3 鉴权测试"
