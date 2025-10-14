#!/bin/bash

# SONiC Custom Services Setup Script
# This script runs as a supervisord service to setup networking and start our custom services

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [SETUP] $1"
}

# Wait for supervisord to be fully ready
log "Waiting for supervisord to be ready..."
sleep 5

# Wait for SONiC services to be ready
log "Checking SONiC service readiness..."
if /opt/sonic-ready-check.sh; then
    log "SONiC system is ready"
else
    log "SONiC readiness check completed with warnings, continuing..."
fi
# SONiC Custom Services Setup Script
# This script runs as a supervisord service to setup networking and start our custom services

log "Starting SONiC custom services setup..."

# Wait a bit for SONiC to initialize
log "Waiting for SONiC initialization..."
sleep 10

# Check if we should setup networking
# SETUP_NETWORK=${SETUP_NETWORK:-true}
# if [ "$SETUP_NETWORK" = "true" ]; then
#     log "Setting up container networking..."
#     if /opt/setup-network.sh; then
#         log "Network setup completed successfully"
#     else
#         log "Network setup failed, continuing anyway..."
#     fi
# fi

# # Wait for supervisord to be fully ready
# log "Waiting for supervisord to be ready..."
# sleep 10

# Services are now configured to auto-start in supervisord
log "Custom services are configured to auto-start via supervisord"
log "You can check their status with: supervisorctl status"

log "Setup completed successfully"

# Exit successfully (this allows supervisord to mark the setup as complete)
exit 0
