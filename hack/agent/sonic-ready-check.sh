#!/bin/bash

# SONiC Service Readiness Check
# This script waits for essential SONiC services to be ready before starting custom services

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SONIC-READY] $1"
}

# Function to check if a service is running
check_service() {
    local service=$1
    supervisorctl status "$service" 2>/dev/null | grep -q "RUNNING"
}

log "Checking SONiC service readiness..."

# Define service tiers based on importance and dependencies
CRITICAL_SERVICES="rsyslogd redis-server gnmi-telemetry switch-proxy-server"

TIMEOUT=120

# Function to wait for a group of services
wait_for_service_group() {
    local group_name=$1
    local services=$2
    local required=${3:-true}
    
    log "Checking $group_name services: $services"
    
    for service in $services; do
        local service_timeout=0
        log "Waiting for $service..."
        
        while [ $service_timeout -lt $TIMEOUT ]; do
            if check_service "$service"; then
                log "✓ $service is running"
                break
            fi
            sleep 2
            service_timeout=$((service_timeout + 2))
            
            if [ $((service_timeout % 20)) -eq 0 ]; then
                log "  Still waiting for $service... (${service_timeout}s)"
            fi
        done
        
        if [ $service_timeout -ge $TIMEOUT ]; then
            if [ "$required" = "true" ]; then
                log "✗ CRITICAL: $service not ready after ${TIMEOUT}s"
                return 1
            else
                log "⚠ WARNING: $service not ready after ${TIMEOUT}s (optional)"
            fi
        fi
    done
    
    return 0
}

# Check critical services first
if ! wait_for_service_group "CRITICAL" "$CRITICAL_SERVICES" true; then
    log "ERROR: Critical services not ready, but continuing anyway..."
fi

# Final status report
log "Service readiness check complete. Final status:"
log "Critical services:"
for service in $CRITICAL_SERVICES; do
    if check_service "$service"; then
        log "  ✓ $service: RUNNING"
    else
        log "  ✗ $service: NOT RUNNING"
    fi
done

log "SONiC system appears to be ready for custom services"
exit 0
