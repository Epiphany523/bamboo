#!/bin/bash

# 停止集群

echo "Stopping Async Task Manager Cluster..."

# 读取并终止进程
for i in 1 2 3; do
    PID_FILE="logs/server-$i.pid"
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            echo "  Stopping server $i (PID: $PID)..."
            kill "$PID"
        else
            echo "  Server $i is not running"
        fi
        rm "$PID_FILE"
    fi
done

echo "Cluster stopped"
