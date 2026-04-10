#!/bin/bash
# AI Gateway 用户管理接口联调测试
# 执行方式: bash test_user_api.sh

set -e

BASE_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "用户管理接口联调测试"
echo "=========================================="

# 颜色函数
pass() { echo -e "${GREEN}✓ $1${NC}"; }
fail() { echo -e "${RED}✗ $1${NC}"; exit 1; }
info() { echo -e "${YELLOW}[INFO] $1${NC}"; }

# 存储测试数据
USER_ID=""
API_KEY=""

# ==========================================
# T1: 创建用户（正常）
# ==========================================
echo -e "\n${YELLOW}[T1] 创建用户（正常）${NC}"
RESP=$(curl -s -X POST "$BASE_URL/admin/api/user/create" \
  -H "Content-Type: application/json" \
  -d '{"email":"test-user@example.com","phone":"13800138001","username":"testuser001"}')
echo "响应: $RESP"

SUCCESS=$(echo "$RESP" | grep -o '"success":true')
if [ -n "$SUCCESS" ]; then
    USER_ID=$(echo "$RESP" | grep -o '"user_id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo "用户ID: $USER_ID"
    pass "T1 创建用户成功"
else
    fail "T1 创建用户失败"
fi

# ==========================================
# T2: 创建用户（邮箱重复，应失败）
# ==========================================
echo -e "\n${YELLOW}[T2] 创建用户（邮箱重复）${NC}"
RESP=$(curl -s -X POST "$BASE_URL/admin/api/user/create" \
  -H "Content-Type: application/json" \
  -d '{"email":"test-user@example.com","phone":"13900139001"}')
echo "响应: $RESP"

ERROR=$(echo "$RESP" | grep -o '"error"')
if [ -n "$ERROR" ]; then
    pass "T2 邮箱重复校验成功"
else
    fail "T2 邮箱重复校验失败"
fi

# ==========================================
# T3: 创建用户（三者都为空，应失败）
# ==========================================
echo -e "\n${YELLOW}[T3] 创建用户（必填校验）${NC}"
RESP=$(curl -s -X POST "$BASE_URL/admin/api/user/create" \
  -H "Content-Type: application/json" \
  -d '{}')
echo "响应: $RESP"

ERROR=$(echo "$RESP" | grep -o '"error"')
if [ -n "$ERROR" ]; then
    pass "T3 至少填一项校验成功"
else
    fail "T3 至少填一项校验失败"
fi

# ==========================================
# T4: 创建用户（邮箱格式错误）
# ==========================================
echo -e "\n${YELLOW}[T4] 创建用户（邮箱格式错误）${NC}"
RESP=$(curl -s -X POST "$BASE_URL/admin/api/user/create" \
  -H "Content-Type: application/json" \
  -d '{"email":"invalid-email"}')
echo "响应: $RESP"

ERROR=$(echo "$RESP" | grep -o '"error"')
if [ -n "$ERROR" ]; then
    pass "T4 邮箱格式校验成功"
else
    fail "T4 邮箱格式校验失败"
fi

# ==========================================
# T5: 创建用户（手机号格式错误）
# ==========================================
echo -e "\n${YELLOW}[T5] 创建用户（手机号格式错误）${NC}"
RESP=$(curl -s -X POST "$BASE_URL/admin/api/user/create" \
  -H "Content-Type: application/json" \
  -d '{"phone":"12345"}')
echo "响应: $RESP"

ERROR=$(echo "$RESP" | grep -o '"error"')
if [ -n "$ERROR" ]; then
    pass "T5 手机号格式校验成功"
else
    fail "T5 手机号格式校验失败"
fi

# ==========================================
# T6: 更新用户
# ==========================================
echo -e "\n${YELLOW}[T6] 更新用户${NC}"
RESP=$(curl -s -X POST "$BASE_URL/admin/api/user/update" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\",\"email\":\"updated@example.com\",\"username\":\"updateduser\"}")
echo "响应: $RESP"

SUCCESS=$(echo "$RESP" | grep -o '"success":true')
if [ -n "$SUCCESS" ]; then
    pass "T6 更新用户成功"
else
    fail "T6 更新用户失败"
fi

# ==========================================
# T7: 为用户创建 API Key
# ==========================================
echo -e "\n${YELLOW}[T7] 为用户创建 API Key${NC}"
RESP=$(curl -s -X POST "$BASE_URL/admin/api/key/create" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\"}")
echo "响应: $RESP"

API_KEY=$(echo "$RESP" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
if [ -n "$API_KEY" ]; then
    echo "API Key: $API_KEY"
    pass "T7 创建 Key 成功"
else
    fail "T7 创建 Key 失败"
fi

# ==========================================
# T8: 用户查询自己的余额
# ==========================================
echo -e "\n${YELLOW}[T8] 用户查询自己的余额${NC}"
RESP=$(curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/me/balance")
echo "响应: $RESP"

SUCCESS=$(echo "$RESP" | grep -o '"success":true')
if [ -n "$SUCCESS" ]; then
    BALANCE=$(echo "$RESP" | grep -o '"balance":[0-9.]*' | cut -d':' -f2)
    echo "余额: $BALANCE"
    pass "T8 查询余额成功"
else
    fail "T8 查询余额失败"
fi

# ==========================================
# T9: 用户查询自己的用量
# ==========================================
echo -e "\n${YELLOW}[T9] 用户查询自己的用量${NC}"
RESP=$(curl -s -H "Authorization: Bearer $API_KEY" \
  "$BASE_URL/v1/me/usage?start=2026-04-01&end=2026-04-30")
echo "响应: $RESP"

SUCCESS=$(echo "$RESP" | grep -o '"success":true')
if [ -n "$SUCCESS" ]; then
    pass "T9 查询用量成功"
else
    fail "T9 查询用量失败"
fi

# ==========================================
# T10: 用无效 Key 查询余额（应失败）
# ==========================================
echo -e "\n${YELLOW}[T10] 无效 Key 鉴权${NC}"
RESP=$(curl -s -H "Authorization: Bearer invalid-key-12345" \
  "$BASE_URL/v1/me/balance")
echo "响应: $RESP"

ERROR=$(echo "$RESP" | grep -o '"error"')
if [ -n "$ERROR" ]; then
    pass "T10 无效 Key 被拒绝"
else
    fail "T10 无效 Key 应被拒绝"
fi

# ==========================================
# T11: 重置 API Key
# ==========================================
echo -e "\n${YELLOW}[T11] 重置 API Key${NC}"
# 获取 key_id
KEY_ID=$(python3 -c "
import sqlite3, json
conn = sqlite3.connect('/mnt/d/aicode/ai-gateway/ai_gateway.db')
cursor = conn.cursor()
cursor.execute(\"SELECT id FROM api_keys WHERE user_id = ?\", ('$USER_ID',))
row = cursor.fetchone()
print(row[0] if row else '')
conn.close()
")
echo "Key ID: $KEY_ID"

RESP=$(curl -s -X POST "$BASE_URL/admin/api/key/reset" \
  -H "Content-Type: application/json" \
  -d "{\"key_id\":\"$KEY_ID\"}")
echo "响应: $RESP"

NEW_KEY=$(echo "$RESP" | grep -o '"api_key":"[^"]*"' | cut -d'"' -f4)
if [ -n "$NEW_KEY" ] && [ "$NEW_KEY" != "$API_KEY" ]; then
    echo "新 Key: $NEW_KEY"
    pass "T11 重置 Key 成功"
else
    fail "T11 重置 Key 失败"
fi

# ==========================================
# T12: 删除用户（含级联）
# ==========================================
echo -e "\n${YELLOW}[T12] 删除用户${NC}"
RESP=$(curl -s -X POST "$BASE_URL/admin/api/user/delete" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER_ID\"}")
echo "响应: $RESP"

SUCCESS=$(echo "$RESP" | grep -o '"success":true')
if [ -n "$SUCCESS" ]; then
    pass "T12 删除用户成功"
else
    fail "T12 删除用户失败"
fi

# 验证删除后 Key 已失效
echo -e "\n${YELLOW}[T12-1] 验证 Key 已失效${NC}"
RESP=$(curl -s -H "Authorization: Bearer $NEW_KEY" \
  "$BASE_URL/v1/me/balance")
echo "响应: $RESP"

ERROR=$(echo "$RESP" | grep -o '"error"')
if [ -n "$ERROR" ]; then
    pass "T12-1 删除后 Key 失效验证成功"
else
    fail "T12-1 删除后 Key 应已失效"
fi

echo -e "\n=========================================="
echo -e "${GREEN}全部测试通过！${NC}"
echo "=========================================="
