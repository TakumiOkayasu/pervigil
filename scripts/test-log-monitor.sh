#!/bin/bash
# log-monitor.sh のテスト（DI方式）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_PASSED=0
TESTS_FAILED=0

assert_contains() {
    local output="$1"
    local expected="$2"
    local test_name="$3"

    if echo "$output" | grep -q "$expected"; then
        echo "✅ PASS: $test_name"
        ((TESTS_PASSED++))
    else
        echo "❌ FAIL: $test_name"
        echo "   Expected to contain: $expected"
        echo "   Got: $output"
        ((TESTS_FAILED++))
    fi
}

assert_exit_code() {
    local actual="$1"
    local expected="$2"
    local test_name="$3"

    if [[ "$actual" -eq "$expected" ]]; then
        echo "✅ PASS: $test_name"
        ((TESTS_PASSED++))
    else
        echo "❌ FAIL: $test_name"
        echo "   Expected exit code: $expected, Got: $actual"
        ((TESTS_FAILED++))
    fi
}

echo "=== log-monitor.sh テスト ==="
echo ""

# テスト1: --status でログパスが表示される
test_status_shows_log_path() {
    echo "[TEST] --status でログパスが表示される"

    local output
    output=$(
        source "$SCRIPT_DIR/log-monitor.sh"
        run_status 2>&1
    )

    assert_contains "$output" "Persistent log:" "--status shows log path label"
}

# テスト2: --status は正常終了する
test_status_exits_successfully() {
    echo "[TEST] --status は正常終了する"

    (
        source "$SCRIPT_DIR/log-monitor.sh"
        run_status >/dev/null 2>&1
    )
    local exit_code=$?

    assert_exit_code "$exit_code" 0 "--status exits with code 0"
}

# テスト3: --test でパターンテストが実行される
test_pattern_test_runs() {
    echo "[TEST] --test でパターンテストが実行される"

    local output
    output=$(
        source "$SCRIPT_DIR/log-monitor.sh"
        run_test 2>&1
    )

    assert_contains "$output" "Testing log patterns" "--test shows testing message"
}

# --- テスト実行 ---

test_status_shows_log_path
echo ""
test_status_exits_successfully
echo ""
test_pattern_test_runs

echo ""
echo "=== 結果: $TESTS_PASSED passed, $TESTS_FAILED failed ==="

[[ "$TESTS_FAILED" -eq 0 ]]
