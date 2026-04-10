#!/bin/bash
# AI Gateway 联调测试 - 运行所有阶段
# 执行方式: bash run_all_phases.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================="
echo "AI Gateway 联调测试 - 全部阶段"
echo "=========================================="

phases=(
    "phase1_startup.sh"
    "phase2_init_user.sh"
    "phase3_auth.sh"
    "phase4_forward.sh"
    "phase5_billing.sh"
    "phase6_admin.sh"
    "phase7_frontend.sh"
    "phase8_logs.sh"
)

for phase in "${phases[@]}"; do
    echo ""
    echo ">>> 运行 $phase <<<"
    bash "$SCRIPT_DIR/$phase"
done

echo ""
echo "=========================================="
echo "全部阶段测试完成!"
echo "=========================================="
