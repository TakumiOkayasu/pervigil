#!/bin/bash
# nic-monitor.sh のテスト（DI方式）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_PASSED=0
TESTS_FAILED=0

# テストヘルパー
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

# --- テストケース ---

echo "=== nic-monitor.sh テスト ==="
echo ""

# テスト1: --test でモック温度が表示される
test_mock_temperature_display() {
    echo "[TEST] --test でモック温度(42°C)が表示される"

    local output
    output=$(
        source "$SCRIPT_DIR/nic-monitor.sh"
        get_nic_temp() { echo "42"; }
        run_test 2>&1
    )

    assert_contains "$output" "42" "--test shows mocked temperature"
}

# テスト2: --status でモック温度とステートが表示される
test_status_shows_temp_and_state() {
    echo "[TEST] --status でモック温度とステートが表示される"

    local output
    output=$(
        source "$SCRIPT_DIR/nic-monitor.sh"
        get_nic_temp() { echo "55"; }
        run_status 2>&1
    )

    assert_contains "$output" "Temperature:" "--status shows Temperature label"
    assert_contains "$output" "55" "--status shows mocked temperature value"
    assert_contains "$output" "State:" "--status shows State label"
}

# テスト3: --status は正常終了する
test_status_exits_successfully() {
    echo "[TEST] --status は正常終了する"

    (
        source "$SCRIPT_DIR/nic-monitor.sh"
        get_nic_temp() { echo "55"; }
        run_status >/dev/null 2>&1
    )
    local exit_code=$?

    assert_exit_code "$exit_code" 0 "--status exits with code 0"
}

# テスト4: 温度取得失敗時にN/Aが表示される
test_temp_unavailable_shows_na() {
    echo "[TEST] 温度取得失敗時にN/Aが表示される"

    local output
    output=$(
        source "$SCRIPT_DIR/nic-monitor.sh"
        # set +eでエラー時も継続させる
        set +e
        get_nic_temp() { echo "N/A"; return 1; }
        run_test 2>&1
    ) || true

    assert_contains "$output" "N/A" "--test shows N/A when temp unavailable"
}

# テスト5: 温度取得失敗時の--statusもN/Aを表示
test_status_shows_na_when_unavailable() {
    echo "[TEST] --status で温度取得失敗時にN/Aが表示される"

    local output
    output=$(
        source "$SCRIPT_DIR/nic-monitor.sh"
        set +e
        get_nic_temp() { echo "N/A"; return 1; }
        run_status 2>&1
    ) || true

    assert_contains "$output" "N/A" "--status shows N/A when temp unavailable"
    assert_contains "$output" "State:" "--status still shows State label"
}

# --- テスト実行 ---

test_mock_temperature_display
echo ""
test_status_shows_temp_and_state
echo ""
test_status_exits_successfully
echo ""
test_temp_unavailable_shows_na
echo ""
test_status_shows_na_when_unavailable

echo ""
echo "=== 結果: $TESTS_PASSED passed, $TESTS_FAILED failed ==="

[[ "$TESTS_FAILED" -eq 0 ]]
