#!/bin/bash

# PolyServer Stress Test Suite
# Tests bandwidth, connections, and performance metrics
# Requires: wrk (install with: sudo apt install build-essential && git clone https://github.com/wg/wrk.git && cd wrk && make)

set -e

TARGET="${1:-http://localhost:9090}"
TEST_DURATION=${2:-60}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if wrk is installed
check_tools() {
    log_info "Checking dependencies..."
    if ! command -v wrk &> /dev/null; then
        log_error "wrk not found. Install with:"
        echo "  git clone https://github.com/wg/wrk.git && cd wrk && make && sudo cp wrk /usr/local/bin/"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_warn "jq not found. Installing..."
        sudo apt update && sudo apt install -y jq
    fi
    
    log_success "All tools available"
}

# Reset metrics before test
reset_metrics() {
    log_info "Resetting server metrics..."
    curl -s -X POST "$TARGET/metrics/reset" || log_warn "Failed to reset metrics"
    sleep 1
}

# Get current metrics from server
get_metrics() {
    curl -s "$TARGET/metrics" 2>/dev/null || echo "{}"
}

# Run baseline test
run_baseline_test() {
    local concurrency=$1
    local test_name="Baseline (${concurrency} connections)"
    
    log_info "Running: $test_name"
    wrk -t 4 \
        -c $concurrency \
        -d ${TEST_DURATION}s \
        --latency \
        "$TARGET/status" \
        2>&1 | tee baseline_${concurrency}.log
    
    log_success "Baseline test completed"
}

# Run ramp-up test
run_ramp_test() {
    log_info "Starting ramp-up stress test..."
    
    for concurrency in 10 50 100 200 500 1000; do
        log_info "Testing with $concurrency concurrent connections..."
        
        reset_metrics
        
        # Run test
        wrk -t 8 \
            -c $concurrency \
            -d ${TEST_DURATION}s \
            --latency \
            "$TARGET/status" \
            2>&1 | tee results_${concurrency}.log
        
        # Capture metrics after test
        metrics=$(get_metrics)
        
        # Print summary
        echo ""
        log_info "Metrics for concurrency=$concurrency:"
        echo "$metrics" | jq '
        {
            active_connections,
            mbps_sent,
            mbps_received,
            avg_latency_ms,
            peak_latency_ms,
            packets_sent,
            car_updates,
            car_failure_rate
        }' 2>/dev/null || echo "$metrics"
        
        echo ""
        sleep 15  # Cool down between tests
    done
    
    log_success "Ramp-up test completed"
}

# Run sustained load test
run_sustained_test() {
    local duration=${1:-600}  # Default 10 minutes
    local concurrency=${2:-500}
    
    log_info "Running sustained load test: ${concurrency} connections for ${duration}s..."
    
    reset_metrics
    
    wrk -t 12 \
        -c $concurrency \
        -d ${duration}s \
        --latency \
        "$TARGET/status" \
        2>&1 | tee sustained_load.log
    
    # Show final metrics
    log_info "Final metrics after sustained test:"
    get_metrics | jq '.' 
    
    log_success "Sustained test completed"
}

# Extract and print results
print_results() {
    log_info "=== TEST RESULTS SUMMARY ==="
    
    echo ""
    log_info "Baseline test results (first successful run):"
    for f in baseline_*.log; do
        if [ -f "$f" ]; then
            echo "File: $f"
            grep -E "Requests/sec:|Avg:|Stdev:|Max:|Body:" "$f" || echo "No results"
            break
        fi
    done
    
    echo ""
    log_info "Ramp-up results by concurrency:"
    for f in results_*.log; do
        if [ -f "$f" ]; then
            concurrency=$(echo $f | grep -oP '\d+' | head -1)
            echo "Concurrency: $concurrency"
            grep "Requests/sec:" "$f" | head -1
        fi
    done
}

# Profile the running server
profile_server() {
    local profile_type=${1:-heap}
    
    log_info "Profiling server ($profile_type)..."
    
    if [ "$profile_type" = "heap" ]; then
        go tool pprof -http=:8080 "http://localhost:6060/debug/pprof/heap" &
    elif [ "$profile_type" = "profile" ]; then
        log_info "Collecting CPU profile for 30 seconds..."
        go tool pprof -http=:8080 "http://localhost:6060/debug/pprof/profile?seconds=30" &
    elif [ "$profile_type" = "goroutine" ]; then
        go tool pprof -http=:8080 "http://localhost:6060/debug/pprof/goroutine" &
    else
        log_error "Unknown profile type: $profile_type"
        return 1
    fi
    
    log_success "Profile started. Browser should open at http://localhost:8080"
}

# Main execution
main() {
    log_info "PolyServer Stress Test Suite"
    log_info "Target: $TARGET"
    log_info "Test duration: ${TEST_DURATION}s"
    echo ""
    
    check_tools
    
    # Parse command line arguments
    case "${3:-ramp}" in
        baseline)
            run_baseline_test 50
            ;;
        ramp)
            run_ramp_test
            ;;
        sustained)
            run_sustained_test "${4:-600}" "${5:-500}"
            ;;
        profile)
            profile_server "${4:-heap}"
            ;;
        *)
            log_error "Unknown test type: ${3:-ramp}"
            echo "Usage: $0 [target_url] [duration] [test_type] [extra_args]"
            echo ""
            echo "Test types:"
            echo "  baseline  - Single baseline test with 50 connections"
            echo "  ramp      - Gradually increase load from 10 to 1000 connections"
            echo "  sustained - Hold heavy load for extended duration"
            echo "  profile   - Start pprof profiler (heap|profile|goroutine)"
            echo ""
            echo "Examples:"
            echo "  $0 http://localhost:9090 60 ramp"
            echo "  $0 http://localhost:9090 30 baseline"
            echo "  $0 http://localhost:9090 600 sustained 600 1000"
            echo "  $0 http://localhost:9090 60 profile heap"
            exit 1
            ;;
    esac
    
    print_results
}

main "$@"
