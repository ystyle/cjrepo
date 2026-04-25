#!/bin/bash
# CJRepo API 测试运行脚本
#
# 使用方式:
#   ./tests/run-tests.sh [port]
#
# 示例:
#   ./tests/run-tests.sh 8060

set -e

PORT=${1:-8060}
BASE_URL="http://localhost:${PORT}"
ADMIN_KEY_MD5="46ab829cae2d14886ec673dd2fea9f26"

echo "=== CJRepo API Tests ==="
echo "Base URL: ${BASE_URL}"
echo ""

# 检查 hurl 是否安装
if ! command -v hurl &> /dev/null; then
    echo "Error: hurl is not installed"
    echo "Install: apt install hurl or brew install hurl"
    exit 1
fi

# 检查服务是否运行
echo "Checking server health..."
if ! curl -s "${BASE_URL}/health" | jq -e '.status == "ok"' > /dev/null 2>&1; then
    echo "Error: Server is not running on port ${PORT}"
    echo "Start server: export CJREPO_ADMIN_KEY=test-admin-key-2024 && ./cjrepo"
    exit 1
fi
echo "Server is healthy ✓"
echo ""

# 运行测试
FAILED=0
PASSED=0

run_test() {
    local test_name=$1
    local test_file=$2
    
    echo "Running: ${test_name}"
    if hurl --variable admin_key=${ADMIN_KEY_MD5} --variable base_url=${BASE_URL} "${test_file}" > /dev/null 2>&1; then
        echo "  ✓ PASSED"
        PASSED=$((PASSED + 1))
    else
        echo "  ✗ FAILED"
        FAILED=$((FAILED + 1))
    fi
}

# 公开 API（无需认证）
run_test "Public API" tests/public-api.hurl

# 认证 API
run_test "Auth API" tests/auth.hurl

# 用户管理
run_test "User Management" tests/user-management.hurl

# 组织管理
run_test "Organization Management" tests/organization.hurl

# 团队权限
run_test "Team Permission" tests/team-permission.hurl

# 包管理
run_test "Package Management" tests/package-management.hurl

# 发布计划
run_test "Publish Plan" tests/publish-plan.hurl

echo ""
echo "=== Test Results ==="
echo "Passed: ${PASSED}"
echo "Failed: ${FAILED}"

if [ ${FAILED} -eq 0 ]; then
    echo "All tests passed! ✓"
    exit 0
else
    echo "Some tests failed. ✗"
    exit 1
fi