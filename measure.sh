#!/bin/bash

# ------------------------
# 配置部分
# ------------------------
URL="https://api.kr777.top"   # 测试目标 URL
TIMES=5                               # 测试次数
PROXY=""                               # 代理，例如 http://127.0.0.1:7890，留空不使用

# ------------------------
# 初始化
# ------------------------
total_tcp=0
total_tls=0
total_ttfb=0
total_total=0

# ------------------------
# 测试循环
# ------------------------
for ((i=1; i<=TIMES; i++)); do
    if [ -z "$PROXY" ]; then
        result=$(curl -o /dev/null -s -w "TCP:%{time_connect} TLS:%{time_appconnect} TTFB:%{time_starttransfer} TOTAL:%{time_total}\n" "$URL")
    else
        result=$(curl -o /dev/null -s -x "$PROXY" -w "TCP:%{time_connect} TLS:%{time_appconnect} TTFB:%{time_starttransfer} TOTAL:%{time_total}\n" "$URL")
    fi

    echo "测量 $i: $result"

    tcp=$(echo "$result" | awk -F 'TCP:' '{print $2}' | awk '{print $1}' | awk -F 'TLS' '{print $1}')
    tls=$(echo "$result" | awk -F 'TLS:' '{print $2}' | awk '{print $1}' | awk -F 'TTFB' '{print $1}')
    ttfb=$(echo "$result" | awk -F 'TTFB:' '{print $2}' | awk '{print $1}' | awk -F 'TOTAL' '{print $1}')
    total=$(echo "$result" | awk -F 'TOTAL:' '{print $2}')

    # 累加
    total_tcp=$(echo "$total_tcp + $tcp" | bc)
    total_tls=$(echo "$total_tls + $tls" | bc)
    total_ttfb=$(echo "$total_ttfb + $ttfb" | bc)
    total_total=$(echo "$total_total + $total" | bc)
done

# ------------------------
# 平均值
# ------------------------
avg_tcp=$(echo "scale=3; $total_tcp / $TIMES" | bc)
avg_tls=$(echo "scale=3; $total_tls / $TIMES" | bc)
avg_ttfb=$(echo "scale=3; $total_ttfb / $TIMES" | bc)
avg_total=$(echo "scale=3; $total_total / $TIMES" | bc)

echo ""
echo "===== 平均延迟 ====="
echo "TCP 建立: $avg_tcp s"
echo "TLS 握手: $avg_tls s"
echo "TTFB: $avg_ttfb s"
echo "总时间: $avg_total s"
