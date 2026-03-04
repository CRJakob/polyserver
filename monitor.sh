#!/bin/bash

# PolyServer Real-Time Monitor
# Run this in a separate terminal while running stress tests
# Shows live bandwidth, connections, and performance metrics

TARGET="${1:-http://localhost:9090}"
INTERVAL=${2:-2}  # Update interval in seconds

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Format bytes to human readable
format_bytes() {
    local bytes=$1
    if [ $bytes -ge 1073741824 ]; then
        echo "$(echo "scale=2; $bytes / 1073741824" | bc)GB"
    elif [ $bytes -ge 1048576 ]; then
        echo "$(echo "scale=2; $bytes / 1048576" | bc)MB"
    elif [ $bytes -ge 1024 ]; then
        echo "$(echo "scale=2; $bytes / 1024" | bc)KB"
    else
        echo "${bytes}B"
    fi
}

# Format seconds to human readable duration
format_duration() {
    local seconds=$1
    if [ $seconds -ge 3600 ]; then
        echo "$((seconds / 3600))h $((seconds % 3600 / 60))m $((seconds % 60))s"
    elif [ $seconds -ge 60 ]; then
        echo "$((seconds / 60))m $((seconds % 60))s"
    else
        echo "${seconds}s"
    fi
}

clear_screen() {
    printf "\033[2J\033[H"
}

draw_bar() {
    local value=$1
    local max=$2
    local width=30
    
    if [ "$max" -gt 0 ] 2>/dev/null; then
        local filled=$(( (value * width) / max ))
        [ $filled -gt $width ] && filled=$width
        [ $filled -lt 0 ] && filled=0
        
        local bar=""
        for ((i=0; i<filled; i++)); do
            bar="${bar}█"
        done
        for ((i=filled; i<width; i++)); do
            bar="${bar}░"
        done
        echo -n "$bar"
    else
        echo -n "░░░░░░░░░░░░░░░░░░░░░░░░░░░░"
    fi
}

show_header() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC} PolyServer Real-Time Monitor @ $(date '+%H:%M:%S')${CYAN}                                    ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════════════════════╝${NC}"
}

show_metrics() {
    local metrics=$1
    
    # Extract metrics
    local uptime=$(echo "$metrics" | jq -r '.uptime_seconds // 0')
    local active=$(echo "$metrics" | jq -r '.active_connections // 0')
    local total=$(echo "$metrics" | jq -r '.total_connections // 0')
    local bytes_sent=$(echo "$metrics" | jq -r '.bytes_sent // 0')
    local bytes_recv=$(echo "$metrics" | jq -r '.bytes_received // 0')
    local mbps_sent=$(echo "$metrics" | jq -r '.mbps_sent // 0')
    local mbps_recv=$(echo "$metrics" | jq -r '.mbps_received // 0')
    local pkt_sent=$(echo "$metrics" | jq -r '.packets_sent // 0')
    local pkt_recv=$(echo "$metrics" | jq -r '.packets_received // 0')
    local avg_pkt_out=$(echo "$metrics" | jq -r '.avg_packet_size_out // 0' | cut -d. -f1)
    local avg_pkt_in=$(echo "$metrics" | jq -r '.avg_packet_size_in // 0' | cut -d. -f1)
    local car_updates=$(echo "$metrics" | jq -r '.car_updates // 0')
    local car_fails=$(echo "$metrics" | jq -r '.car_update_failures // 0')
    local failure_rate=$(echo "$metrics" | jq -r '.car_failure_rate // 0' | cut -d. -f1)
    local avg_latency=$(echo "$metrics" | jq -r '.avg_latency_ms // 0' | cut -d. -f1)
    local peak_latency=$(echo "$metrics" | jq -r '.peak_latency_ms // 0' | cut -d. -f1)
    local pings=$(echo "$metrics" | jq -r '.pings_processed // 0')
    local joined=$(echo "$metrics" | jq -r '.players_joined // 0')
    local left=$(echo "$metrics" | jq -r '.players_left // 0')
    
    # Color based on values
    local conn_color=$GREEN
    [ $active -gt 500 ] && conn_color=$YELLOW
    [ $active -gt 1000 ] && conn_color=$RED
    
    local latency_color=$GREEN
    [ $avg_latency -gt 100 ] && latency_color=$YELLOW
    [ $avg_latency -gt 200 ] && latency_color=$RED
    
    local failure_color=$GREEN
    [ $failure_rate -gt 1 ] && failure_color=$YELLOW
    [ $failure_rate -gt 5 ] && failure_color=$RED
    
    # Build output
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  SYSTEM OVERVIEW${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    printf "  Uptime: %-20s Active Connections: ${conn_color}%d${NC} / Total: %d\n" \
        "$(format_duration $(echo "$uptime" | cut -d. -f1))" "$active" "$total"
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  BANDWIDTH${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    
    printf "  Out: %.2f Mbps  $(draw_bar $(echo "$mbps_sent" | cut -d. -f1) 100)  ($(format_bytes $bytes_sent))\n" "$mbps_sent"
    printf "  In:  %.2f Mbps  $(draw_bar $(echo "$mbps_recv" | cut -d. -f1) 100)  ($(format_bytes $bytes_recv))\n" "$mbps_recv"
    printf "  Avg Packet Size: Out=%d bytes, In=%d bytes\n" "$avg_pkt_out" "$avg_pkt_in"
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  PACKETS & MESSAGES${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    printf "  Sent: %-15d Received: %-15d\n" "$pkt_sent" "$pkt_recv"
    printf "  Car Updates: %-15d (Failures: %d, Rate: ${failure_color}%.1f%%${NC})\n" "$car_updates" "$car_fails" "$failure_rate"
    printf "  Pings: %d\n" "$pings"
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  PERFORMANCE${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    printf "  Avg Latency: ${latency_color}%-6d ms${NC}  Peak: %-6d ms\n" "$avg_latency" "$peak_latency"
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  PLAYERS${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════${NC}"
    printf "  Joined: %-15d Left: %d  Net: +%d\n" "$joined" "$left" "$((joined - left))"
    
    echo ""
}

monitor_loop() {
    last_bytes_sent=0
    last_bytes_recv=0
    
    while true; do
        clear_screen
        show_header
        
        # Fetch metrics with timeout
        metrics=$(timeout 5 curl -s "$TARGET/metrics" 2>/dev/null || echo '{}')
        
        if [ -z "$metrics" ] || [ "$metrics" == "{}" ]; then
            echo ""
            echo -e "${RED}✗ Cannot reach server at $TARGET${NC}"
            echo ""
            echo "Make sure the server is running:"
            echo "  cd ~/polyserver-go"
            echo "  go run . -port 8091"
            echo ""
        else
            show_metrics "$metrics"
        fi
        
        echo -e "${CYAN}Refresh interval: ${INTERVAL}s (Press Ctrl+C to exit)${NC}"
        sleep $INTERVAL
    done
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "jq is required. Installing..."
    sudo apt update && sudo apt install -y jq
fi

echo "Starting real-time monitor for $TARGET (interval: ${INTERVAL}s)"
echo "Press Ctrl+C to exit"
sleep 2

monitor_loop
