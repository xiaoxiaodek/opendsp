#!/bin/bash
# 多广告创意量级 单实例最大QPS 压测脚本
# 采用二分查找逐步提升 QPS 直到错误率 > 1%
set -e

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJhZHZlcnRpc2VyX2lkIjowLCJyb2xlIjoiYWRtaW4ifQ.qIlocv4NXNYs5APHVdQFiCs1BKA3BBwNIlDenU5GV7w"
ADMANAGER="http://localhost:8081"
ADSERVER="http://localhost:8080"
BASE_DIR=$(cd "$(dirname "$0")/.." && pwd)
RESULT_FILE="$BASE_DIR/scripts/stress_results.csv"
MOCKADX_CFG="$BASE_DIR/scripts/mockadx_stress.yaml"
TEST_DURATION=10

# 测试规模: 广告组数 (每广告组1个创意)
SCALES=("1" "10" "50" "100" "200" "500")

# ---- 辅助函数 ----
write_config() {
    local qps=$1
    cat > "$MOCKADX_CFG" <<EOF
protocol: iqiyi
target: $ADSERVER
endpoint: /rtb/iqiyi
duration: ${TEST_DURATION}s
qps: $qps
concurrency: 200
gzip: true
timeout: 500ms
scenario:
  profile: mixed
  mixed_weights:
    hot-user: 0.2
    long-tail: 0.7
    peak: 0.1
  hot_user_pool: 100
  long_tail_ids: 10000000
funnel:
  win_rate: 0.30
  imp_rate: 0.95
  click_rate: 0.01
  conv_rate: 0.001
receiver:
  listen: :9095
  metrics_path: /metrics
report:
  interval: 1s
  output: /tmp/stress_report.json
EOF
}

create_ad_group() {
    local idx=$1
    curl -s -X POST "$ADMANAGER/api/v1/campaigns/6/adgroups" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{
            \"name\": \"压测广告组-${idx}\",
            \"bidType\": 1,
            \"bidPrice\": 30.0,
            \"dailyBudget\": 10000.0,
            \"freqCap\": 0,
            \"targeting\": \"{\\\"inventory\\\":{\\\"media\\\":[\\\"iqiyi\\\"]}}\"
        }" | jq -r '.id // empty'
}

activate_ad_group() {
    local ag_id=$1
    curl -s -X PATCH "$ADMANAGER/api/v1/adgroups/${ag_id}/status" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"status": 1}' > /dev/null
}

create_creative() {
    local ag_id=$1
    local idx=$2
    curl -s -X POST "$ADMANAGER/api/v1/adgroups/${ag_id}/creatives" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{
            \"name\": \"压测创意-${idx}\",
            \"creativeType\": 1,
            \"assetUrl\": \"https://example.com/banner.jpg\",
            \"assetWidth\": 300,
            \"assetHeight\": 250,
            \"assetMime\": \"image/jpeg\",
            \"title\": \"压测创意-${idx}\",
            \"description\": \"压测用创意\",
            \"landingUrl\": \"https://example.com/landing\"
        }" > /dev/null
}

run_mockadx_with_json() {
    local qps=$1
    write_config "$qps"
    rm -f /tmp/stress_report.json
    cd "$BASE_DIR" && go run ./cmd/mockadx -config "$MOCKADX_CFG" 2>&1 | tail -3
    cat /tmp/stress_report.json 2>/dev/null || echo '{"summary":{"total_requests":0,"total_bids":0,"avg_qps":0,"latency_p50":0,"latency_p95":0,"latency_p99":0,"status_distribution":{"error":999999}}}'
}

# 判断是否通过: 错误率 < 1%
is_passing() {
    local json=$1
    local total=$(echo "$json" | jq -r '.summary.total_requests')
    local errors=$(echo "$json" | jq -r '.summary.status_distribution.error // 0')
    if [ "$total" -eq 0 ]; then return 1; fi
    local err_rate=$(echo "scale=4; $errors / $total" | bc)
    [ "$(echo "$err_rate < 0.01" | bc)" -eq 1 ]
}

# 二分查找最大 QPS
find_max_qps() {
    local low=100
    local high=10000
    local best_qps=0
    local best_json=""
    local step=100

    echo "  [二分查找] range=[$low, $high] step=$step"

    while [ $low -le $high ]; do
        local mid=$(( (low + high) / 2 ))
        mid=$(( mid / step * step ))
        [ "$mid" -lt "$low" ] && mid=$low

        echo -n "    QPS=$mid ... "
        local json=$(run_mockadx_with_json "$mid")

        if is_passing "$json"; then
            best_qps=$mid
            best_json="$json"
            low=$((mid + step))
            local p50=$(echo "$json" | jq -r '.summary.latency_p50')
            local p95=$(echo "$json" | jq -r '.summary.latency_p95')
            local p99=$(echo "$json" | jq -r '.summary.latency_p99')
            echo "PASS (p50=$(printf "%.1f" "$(echo "$p50*1000"|bc)")ms)"
        else
            high=$((mid - step))
            local err=$(echo "$json" | jq -r '.summary.status_distribution.error // 0')
            echo "FAIL (errors=$err)"
        fi
    done

    echo "$best_qps"$'\n'"$best_json"
}

# ---- 清理 ----
cleanup() {
    echo "清理旧压测数据..."
    docker exec opendsp-postgres-1 psql -U opendsp -d opendsp -c "DELETE FROM creative WHERE name LIKE '压测创意-%';" 2>/dev/null || true
    docker exec opendsp-postgres-1 psql -U opendsp -d opendsp -c "DELETE FROM ad_group WHERE name LIKE '压测广告组-%';" 2>/dev/null || true
}

rebuild_index() {
    echo "  重启 ad-server 触发索引重建..."
    docker restart opendsp-ad-server-1 > /dev/null 2>&1
    sleep 8
    local idx_line=$(docker logs opendsp-ad-server-1 --tail 5 2>&1 | grep "index built" || echo "unknown")
    echo "  $idx_line"
}

# ---- 主流程 ----
echo "=============================================="
echo "  多创意量级 单实例最大QPS 压测"
echo "  时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "  时长: ${TEST_DURATION}s/轮"
echo "=============================================="
echo ""

echo "scale,ad_groups,creatives,max_qps,p50_ms,p95_ms,p99_ms,total_requests,errors,bids" > "$RESULT_FILE"

cleanup

for scale in "${SCALES[@]}"; do
    echo ""
    echo "========== 规模: ${scale} 广告组 (${scale} 创意) =========="

    # 批量创建
    echo "  创建中..."
    ag_ids=()
    for i in $(seq 1 "$scale"); do
        ag_id=$(create_ad_group "$i")
        if [ -n "$ag_id" ] && [ "$ag_id" != "null" ]; then
            ag_ids+=("$ag_id")
            create_creative "$ag_id" "$i" &
        fi
        # 每批 20 个并发, 控制创建速度
        if [ $((i % 20)) -eq 0 ]; then
            wait
            echo "    已创建 $i/$scale..."
        fi
    done
    wait
    echo "  创建完成: ${#ag_ids[@]} 个广告组"

    # 激活所有广告组
    echo "  激活广告组..."
    for ag_id in "${ag_ids[@]}"; do
        activate_ad_group "$ag_id" &
    done
    wait
    echo "  激活完成"

    rebuild_index

    # 二分查找
    result=$(find_max_qps)
    max_qps=$(echo "$result" | head -1)
    json=$(echo "$result" | tail -n +2)

    if [ "$max_qps" -gt 0 ]; then
        p50=$(echo "$json" | jq -r '.summary.latency_p50')
        p95=$(echo "$json" | jq -r '.summary.latency_p95')
        p99=$(echo "$json" | jq -r '.summary.latency_p99')
        total=$(echo "$json" | jq -r '.summary.total_requests')
        errors=$(echo "$json" | jq -r '.summary.status_distribution.error // 0')
        bids=$(echo "$json" | jq -r '.summary.total_bids')
        p50_ms=$(printf "%.1f" "$(echo "$p50*1000" | bc)")
        p95_ms=$(printf "%.1f" "$(echo "$p95*1000" | bc)")
        p99_ms=$(printf "%.1f" "$(echo "$p99*1000" | bc)")

        echo ""
        echo "  ★ MAX QPS = $max_qps (p50=${p50_ms}ms p95=${p95_ms}ms p99=${p99_ms}ms)"
        echo "$scale,$scale,$scale,$max_qps,$p50_ms,$p95_ms,$p99_ms,$total,$errors,$bids" >> "$RESULT_FILE"
    else
        echo "  ★ 未找到可用 QPS"
    fi
done

echo ""
echo "=============================================="
echo "  压测完成! 结果: $RESULT_FILE"
echo "=============================================="
echo ""
column -t -s, "$RESULT_FILE"