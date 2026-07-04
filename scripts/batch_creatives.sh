#!/bin/bash
# 批量创建 100 个创意, 每个创意对应一个广告组
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJhZHZlcnRpc2VyX2lkIjowLCJyb2xlIjoiYWRtaW4ifQ.qIlocv4NXNYs5APHVdQFiCs1BKA3BBwNIlDenU5GV7w"

AG_IDS=$(docker exec opendsp-postgres-1 psql -U opendsp -d opendsp -t -A -c "SELECT id FROM ad_group WHERE name LIKE '压测广告组-%' ORDER BY id;")

idx=1
total=$(echo "$AG_IDS" | wc -l | tr -d ' ')
echo "Creating $total creatives..."

for ag_id in $AG_IDS; do
    resp=$(curl -s -X POST "http://localhost:8081/api/v1/adgroups/${ag_id}/creatives" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"name\":\"压测创意-${idx}\",\"creativeType\":1,\"assetUrl\":\"https://example.com/banner.jpg\",\"assetWidth\":300,\"assetHeight\":250,\"assetMime\":\"image/jpeg\",\"title\":\"压测创意-${idx}\",\"description\":\"压测\",\"landingUrl\":\"https://example.com\"}")
    cr_id=$(echo "$resp" | jq -r '.id // "FAIL"')
    if [ "$cr_id" = "FAIL" ] || [ -z "$cr_id" ]; then
        echo "FAIL: ag_id=$ag_id idx=$idx resp=$resp"
    fi
    idx=$((idx+1))
    if [ $((idx % 10)) -eq 1 ]; then
        echo "  $((idx-1))/$total done..."
    fi
done

echo "Done. Verifying..."
docker exec opendsp-postgres-1 psql -U opendsp -d opendsp -c "SELECT COUNT(*) creatives FROM creative WHERE name LIKE '压测创意-%';"