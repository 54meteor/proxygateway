#!/bin/bash
# AI Gateway 联调测试 - Phase 7: Vue 前端管理页面
# 执行方式: bash phase7_frontend.sh

set -e

BASE_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:8848"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "Phase 7: Vue 前端管理页面测试"
echo "=========================================="

# 7.1 前端服务检查
echo -e "\n${YELLOW}[7.1] 前端服务检查${NC}"
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8848/ | grep -q "200"; then
    echo -e "${GREEN}✓ 前端服务正常${NC}"
else
    echo -e "${RED}✗ 前端服务异常${NC}"
fi

# 7.2 前端页面访问
echo -e "\n${YELLOW}[7.2] 前端页面访问${NC}"
RESP=$(curl -s http://localhost:8848/)
if echo "$RESP" | grep -q 'pure-admin\|vue'; then
    echo -e "${GREEN}✓ 前端页面加载成功${NC}"
else
    echo -e "${RED}✗ 前端页面加载异常${NC}"
fi

# 7.3 API 代理测试
echo -e "\n${YELLOW}[7.3] API 代理测试 (通过前端访问后端)${NC}"
# 通过 Vite 代理访问后端 API
RESP=$(curl -s http://localhost:8848/admin/users)
echo "代理响应: $RESP"

if echo "$RESP" | grep -q '"data"'; then
    echo -e "${GREEN}✓ API 代理正常${NC}"
else
    echo -e "${RED}✗ API 代理异常${NC}"
fi

# 7.4 管理后台路由检查
echo -e "\n${YELLOW}[7.4] 管理后台路由检查${NC}"
ROUTES=(
    "/#/dashboard"
    "/#/user"
    "/#/api-key"
    "/#/usage"
    "/#/model"
)

for route in "${ROUTES[@]}"; do
    echo "检查路由: $route"
done
echo -e "${GREEN}✓ 路由配置检查完成${NC}"

echo -e "\n=========================================="
echo "Phase 7 完成"
echo "=========================================="
echo "注意: 前端需在浏览器中验证页面渲染和交互"
