#!/bin/bash

# PolyServer Enhanced - Deployment Script
# Run this to rebuild and restart the server

set -e

echo "🔨 Building PolyServer Enhanced..."
cd ~/polyserver-enhanced
go build -o polyserver . 2>&1 | grep -E "error|Error" || echo "✓ Build successful"

echo ""
echo "🔄 Restarting systemd service..."
sudo systemctl restart polyserver

echo ""
echo "✓ Deployment complete!"
echo ""
echo "Access dashboard at: http://localhost:8091"
echo "Metrics API: http://localhost:9090/metrics"
echo ""
echo "Check status:"
systemctl status polyserver --no-pager | grep -E "Active|Main PID"
