#!/bin/bash

# Production startup script for WebSocket Health Monitor
# This script ensures continuous operation with proper logging and monitoring

set -euo pipefail

# Configuration
PROJECT_DIR="/home/sigma/Projects/payer-status-io"
CONFIG_FILE="${PROJECT_DIR}/docs/payer_status.yaml"
LOG_DIR="${PROJECT_DIR}/logs"
PID_FILE="${PROJECT_DIR}/health-monitor.pid"
BINARY_NAME="health-monitor"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS:${NC} $1"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

# Create necessary directories
setup_directories() {
    log "Setting up directories..."
    mkdir -p "$LOG_DIR"
    mkdir -p "${PROJECT_DIR}/bin"
}

# Check if configuration file exists and is valid
validate_config() {
    log "Validating configuration..."
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        error "Configuration file not found: $CONFIG_FILE"
        exit 1
    fi
    
    # Check YAML syntax
    if ! python3 -c "import yaml; yaml.safe_load(open('$CONFIG_FILE'))" 2>/dev/null; then
        error "Invalid YAML syntax in configuration file"
        exit 1
    fi
    
    # Count payers and endpoints
    local payer_count=$(grep -c "^  - name:" "$CONFIG_FILE" || echo "0")
    local endpoint_count=$(grep -c "    - type:" "$CONFIG_FILE" || echo "0")
    
    success "Configuration validated: $payer_count payers, $endpoint_count endpoints"
}

# Build the application
build_app() {
    log "Building application..."
    cd "$PROJECT_DIR"
    
    if ! go build -o "bin/$BINARY_NAME" cmd/server/main.go; then
        error "Failed to build application"
        exit 1
    fi
    
    success "Application built successfully"
}

# Check if the application is already running
check_running() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            return 0  # Running
        else
            rm -f "$PID_FILE"
            return 1  # Not running
        fi
    fi
    return 1  # Not running
}

# Stop the application
stop_app() {
    if check_running; then
        local pid=$(cat "$PID_FILE")
        log "Stopping application (PID: $pid)..."
        
        # Send SIGTERM for graceful shutdown
        kill -TERM "$pid" 2>/dev/null || true
        
        # Wait for graceful shutdown
        local count=0
        while kill -0 "$pid" 2>/dev/null && [[ $count -lt 30 ]]; do
            sleep 1
            ((count++))
        done
        
        # Force kill if still running
        if kill -0 "$pid" 2>/dev/null; then
            warn "Graceful shutdown failed, force killing..."
            kill -KILL "$pid" 2>/dev/null || true
        fi
        
        rm -f "$PID_FILE"
        success "Application stopped"
    else
        log "Application is not running"
    fi
}

# Start the application
start_app() {
    if check_running; then
        warn "Application is already running (PID: $(cat "$PID_FILE"))"
        return 0
    fi
    
    log "Starting WebSocket Health Monitor..."
    
    # Set environment variables
    export CONFIG_PATH="$CONFIG_FILE"
    export GOMAXPROCS=$(nproc)
    
    # Start the application in background
    cd "$PROJECT_DIR"
    nohup "./bin/$BINARY_NAME" \
        > "$LOG_DIR/health-monitor.log" 2>&1 &
    
    local pid=$!
    echo "$pid" > "$PID_FILE"
    
    # Wait a moment and check if it's still running
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        success "Application started successfully (PID: $pid)"
        log "Logs: $LOG_DIR/health-monitor.log"
        log "WebSocket: ws://localhost:8080/ws"
        log "Metrics: http://localhost:9090/metrics"
        log "Dashboard: http://localhost:8080/"
    else
        error "Application failed to start"
        rm -f "$PID_FILE"
        exit 1
    fi
}

# Restart the application
restart_app() {
    log "Restarting application..."
    stop_app
    sleep 2
    start_app
}

# Reload configuration (send SIGHUP)
reload_config() {
    if check_running; then
        local pid=$(cat "$PID_FILE")
        log "Reloading configuration (sending SIGHUP to PID: $pid)..."
        kill -HUP "$pid"
        success "Configuration reload signal sent"
    else
        error "Application is not running, cannot reload configuration"
        exit 1
    fi
}

# Show application status
show_status() {
    if check_running; then
        local pid=$(cat "$PID_FILE")
        success "Application is running (PID: $pid)"
        
        # Show resource usage
        if command -v ps >/dev/null; then
            log "Resource usage:"
            ps -p "$pid" -o pid,ppid,pcpu,pmem,etime,cmd --no-headers || true
        fi
        
        # Show recent logs
        if [[ -f "$LOG_DIR/health-monitor.log" ]]; then
            log "Recent logs (last 10 lines):"
            tail -n 10 "$LOG_DIR/health-monitor.log"
        fi
    else
        warn "Application is not running"
    fi
}

# Monitor the application (keep it running)
monitor_app() {
    log "Starting monitoring mode (Ctrl+C to stop)..."
    
    trap 'log "Monitoring stopped"; exit 0' INT TERM
    
    while true; do
        if ! check_running; then
            warn "Application is not running, restarting..."
            start_app
        fi
        
        # Check every 30 seconds
        sleep 30
    done
}

# Show help
show_help() {
    cat << EOF
WebSocket Health Monitor - Production Control Script

Usage: $0 [COMMAND]

Commands:
    start       Start the application
    stop        Stop the application
    restart     Restart the application
    reload      Reload configuration (SIGHUP)
    status      Show application status
    monitor     Start monitoring mode (auto-restart)
    build       Build the application
    validate    Validate configuration
    logs        Show recent logs
    help        Show this help message

Files:
    Config:     $CONFIG_FILE
    Logs:       $LOG_DIR/health-monitor.log
    PID:        $PID_FILE
    Binary:     ${PROJECT_DIR}/bin/$BINARY_NAME

Endpoints:
    WebSocket:  ws://localhost:8080/ws
    Metrics:    http://localhost:9090/metrics
    Dashboard:  http://localhost:8080/
    Health:     http://localhost:8080/health

EOF
}

# Show logs
show_logs() {
    if [[ -f "$LOG_DIR/health-monitor.log" ]]; then
        log "Showing logs (press Ctrl+C to exit):"
        tail -f "$LOG_DIR/health-monitor.log"
    else
        warn "Log file not found: $LOG_DIR/health-monitor.log"
    fi
}

# Main script logic
main() {
    case "${1:-help}" in
        start)
            setup_directories
            validate_config
            build_app
            start_app
            ;;
        stop)
            stop_app
            ;;
        restart)
            setup_directories
            validate_config
            build_app
            restart_app
            ;;
        reload)
            validate_config
            reload_config
            ;;
        status)
            show_status
            ;;
        monitor)
            setup_directories
            validate_config
            build_app
            start_app
            monitor_app
            ;;
        build)
            setup_directories
            build_app
            ;;
        validate)
            validate_config
            ;;
        logs)
            show_logs
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
