#!/bin/bash

# Simple network setup script
# Brings up eth0 and adds default route

# set -e

# INTERFACE=${1:-eth0}

# # Bring up the interface
# echo "Bringing up interface $INTERFACE..."
# ip link set $INTERFACE up

# # Wait for interface to be ready
# sleep 2

# # Get the IP address and calculate gateway (assumes .1 gateway)
# IP_INFO=$(ip addr show $INTERFACE | grep "inet " | head -n1 | awk '{print $2}')

# if [ -z "$IP_INFO" ]; then
#     echo "ERROR: No IP address found for $INTERFACE"
#     exit 1
# fi

# sleep 2

# # Extract network and assume gateway is .1
# GATEWAY=$(echo $IP_INFO | cut -d'/' -f1 | sed 's/\.[0-9]*$/\.1/')

# echo "Interface $INTERFACE IP: $IP_INFO"
# echo "Using gateway: $GATEWAY"

# # Remove existing default route if any
# ip route del default 2>/dev/null || true

# # Add default route
# echo "Adding default route..."
# ip route add default via $GATEWAY dev $INTERFACE

# echo "Network setup complete!"
# echo "Current routes:"
# ip route show

set -e

INTERFACE=${1:-eth0}
CHECK_INTERVAL=${2:-1}  # Check every 1 second by default
LOG_PREFIX="[NetworkDaemon]"

echo "$LOG_PREFIX Starting network monitoring daemon for $INTERFACE (checking every ${CHECK_INTERVAL}s)"

# Function to check and restore network
check_and_restore_network() {
    local interface=$1
    
    # Check if interface is up
    if ! ip link show "$interface" | grep -q "state UP"; then
        echo "$LOG_PREFIX Interface $interface is down, bringing it up..."
        ip link set "$interface" up
        sleep 1
    fi
    
    # Check if default route exists
    if ! ip route show default | grep -q "dev $interface"; then
        echo "$LOG_PREFIX Default route missing for $interface, restoring..."
        
        # Get the IP address and calculate gateway
        IP_INFO=$(ip addr show "$interface" | grep "inet " | head -n1 | awk '{print $2}' 2>/dev/null)
        
        if [ -z "$IP_INFO" ]; then
            echo "$LOG_PREFIX WARNING: No IP address found for $interface"
            return 1
        fi
        
        # Extract network and assume gateway is .1
        GATEWAY=$(echo "$IP_INFO" | cut -d'/' -f1 | sed 's/\.[0-9]*$/\.1/')
        
        echo "$LOG_PREFIX Restoring route: default via $GATEWAY dev $interface"
        
        # Remove existing default route if any (silently)
        ip route del default 2>/dev/null || true
        
        # Add default route
        if ip route add default via "$GATEWAY" dev "$interface" 2>/dev/null; then
            echo "$LOG_PREFIX Route restored successfully"
        else
            echo "$LOG_PREFIX Failed to restore route"
            return 1
        fi
    fi
    
    return 0
}

# Main monitoring loop
while true; do
    if ! check_and_restore_network "$INTERFACE"; then
        echo "$LOG_PREFIX Network check failed, will retry in ${CHECK_INTERVAL}s"
    fi
    
    sleep "$CHECK_INTERVAL"
done


