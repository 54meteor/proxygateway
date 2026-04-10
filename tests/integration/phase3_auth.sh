#!/bin/bash
# AI Gateway 联调测试 - Phase 3: 鉴权流程
# 执行方式: bash phase3_auth.sh

set -e

BASE_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 加载 Phase 2 保存的 API Key
if [ -f /tmp/test_api_key.txt ]; then
    source /tmp/test_api_key.txt
else
    echo -e "${RED}请先运行 Phase 2 初始化用户${NC}"
    exit 1
fi

echo "=========================================="
echo "Phase 3: 鉴权流程测试"
echo "=========================================="

# 3.1 正确 Key 请求
echo -e "\n${YELLOW}[3.1] 正确 Key 请求 /v1/chat/completions${NC}"
RESP=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"model":"MiniMax-M2.5","messages":[{"role":"user","content":"hi"}]}')
echo "响应: $RESP"

if echo "$RESP" | grep -qi '"Choices"'; then
    echo -e "${GREEN}✓ 正确 Key 请求通过${NC}"
else
    echo -e "${RED}✗ 正确 Key 请求失败${NC}"
fi

# 3.2 错误 Key 请求
echo -e "\n${YELLOW}[3.2] 错误 Key 请求${NC}"
RESP=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
    -H "Authorization: Bearer invalid-key-12345" \
    -H "Content-Type: application/json" \
    -d '{"model":"MiniMax-M2.5","messages":[{"role":"user","content":"hi"}]}')
echo "响应: $RESP"

if echo "$RESP" | grep -q '"error"'; then
    echo -e "${GREEN}✓ 错误 Key 被正确拒绝${NC}"
else
    echo -e "${RED}✗ 错误 Key 未被拒绝${NC}"
fi

# 3.3 缺失 Key 请求
echo -e "\n${YELLOW}[3.3] 缺失 Key 请求${NC}"
RESP=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{"model":"MiniMax-M2.5","messages":[{"role":"user","content":"hi"}]}')
echo "响应: $RESP"

if echo "$RESP" | grep -q '"error"'; then
    echo -e "${GREEN}✓ 缺失 Key 被正确拒绝${NC}"
else
    echo -e "${RED}✗ 缺失 Key 未被拒绝${NC}"
fi

echo -e "\n=========================================="
echo "Phase 3 完成"
echo "=========================================="
